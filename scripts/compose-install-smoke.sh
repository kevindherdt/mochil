#!/bin/sh
set -eu

project="bobobooks-install-smoke-$$"
port="${BOBOBOOKS_SMOKE_PORT:-18101}"
page="$(mktemp)"
settings="$(mktemp)"
config_root="$(mktemp -d)"

cleanup() {
  MOCHIL_CONFIG_PATH="$config_root" MOCHIL_PORT="$port" docker compose --env-file /dev/null -f compose.yaml \
    -p "$project" down -v >/dev/null 2>&1 || true
  rm -f "$page" "$settings"
  rm -rf "$config_root"
}
trap cleanup EXIT INT TERM

MOCHIL_CONFIG_PATH="$config_root" MOCHIL_PORT="$port" docker compose --env-file /dev/null -f compose.yaml \
  -p "$project" up -d --build

attempt=0
while [ "$attempt" -lt 30 ]; do
  if curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null &&
     curl -fsS "http://127.0.0.1:$port/" -o "$page" &&
     grep -q '<title>BoboBooks</title>' "$page" &&
     grep -q 'id="settings-view"' "$page"; then
    container_id="$(MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" ps -q mochil)"
    restart_count="$(docker inspect --format '{{.RestartCount}}' "$container_id")"
    if [ "$restart_count" -ne 0 ]; then
      printf '%s\n' "Clean Compose installation restarted $restart_count time(s)." >&2
      exit 1
    fi
    if [ ! -s "$config_root/config.json" ]; then
      printf '%s\n' "Clean Compose installation did not persist config.json in the host config directory." >&2
      exit 1
    fi
    runtime_uid="$(docker exec "$container_id" awk '/^Uid:/{print $2}' /proc/1/status)"
    if [ "$runtime_uid" -ne 1000 ]; then
      printf '%s\n' "BoboBooks did not drop to its unprivileged runtime user." >&2
      exit 1
    fi
    printf '%s\n' "Clean Compose installation is healthy and serves first-run setup on port $port."
    break
  fi
  attempt=$((attempt + 1))
  sleep 1
done

if [ "$attempt" -ge 30 ]; then
  MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" logs --tail=100
  printf '%s\n' "Clean Compose installation did not become healthy." >&2
  exit 1
fi

# A removed share is a recoverable library error, not an application failure.
# Exercise the exact upgrade case where persisted settings still name storage
# that is no longer mounted on the host.
container_id="$(MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" ps -q mochil)"
docker cp deployment/install-test-missing-library.json "$container_id:/config/config.json"
# Reproduce a restored/private config owned by the host user after an LXC
# restart. Startup must repair ownership without exposing stored secrets.
docker exec -u 0 "$container_id" sh -c 'chown 1000:1000 /config/config.json && chmod 0600 /config/config.json'
MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" restart mochil >/dev/null

attempt=0
while [ "$attempt" -lt 30 ]; do
  if curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null &&
     curl -fsS "http://127.0.0.1:$port/" -o "$page" &&
     curl -fsS "http://127.0.0.1:$port/api/settings" -o "$settings" &&
     grep -q '<title>BoboBooks</title>' "$page" &&
     grep -q 'id="settings-view"' "$page" &&
     grep -Eq '"id"[[:space:]]*:[[:space:]]*"removed-library"' "$settings"; then
    container_id="$(MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" ps -q mochil)"
    restart_count="$(docker inspect --format '{{.RestartCount}}' "$container_id")"
    if [ "$restart_count" -eq 0 ]; then
      config_mode="$(docker exec "$container_id" stat -c '%u:%g:%a' /config/config.json)"
      if [ "$config_mode" != '1000:1000:600' ]; then
        printf '%s\n' "Restored config ownership or privacy was not preserved: $config_mode" >&2
        exit 1
      fi
      printf '%s\n' "Persisted settings with a missing library remain visible and repairable in the web app."
      exit 0
    fi
  fi
  attempt=$((attempt + 1))
  sleep 1
done

MOCHIL_CONFIG_PATH="$config_root" docker compose --env-file /dev/null -f compose.yaml -p "$project" logs --tail=100
printf '%s\n' "Installation did not recover from a missing configured library." >&2
exit 1
