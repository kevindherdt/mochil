#!/bin/sh
# Proves a fresh install works: build the image, boot it against empty volumes, and check that
# someone who has never seen this project gets a usable web UI with no config file, no accounts,
# no libraries and no environment variables. That is the contract "easy to install" means here.
#
# Usage: scripts/install-smoke.sh [--no-build]
#   IMAGE=...  tag to build/run (default mochil:install-smoke)
#   PORT=...   host port to publish (default 18099)
set -eu

cd "$(dirname "$0")/.."

image="${IMAGE:-mochil:install-smoke}"
port="${PORT:-18099}"
name="mochil-install-smoke-$$"
build=1
[ "${1:-}" = "--no-build" ] && build=0

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
	docker volume rm -f "$name-config" "$name-data" "$name-books" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

fail() {
	echo "FAIL: $*" >&2
	echo "--- container logs ---" >&2
	docker logs "$name" >&2 2>&1 || true
	exit 1
}

if [ "$build" = 1 ]; then
	echo "== building $image"
	docker build -q -f Containerfile -t "$image" . >/dev/null || fail "image did not build"
fi

echo "== booting with empty volumes and no configuration"
docker volume create "$name-config" >/dev/null
docker volume create "$name-data" >/dev/null
docker volume create "$name-books" >/dev/null
docker run -d --name "$name" -p "127.0.0.1:$port:8080" \
	-v "$name-config:/config" -v "$name-data:/data" -v "$name-books:/books" \
	"$image" >/dev/null || fail "container did not start"

echo -n "== waiting for /healthz"
ready=0
i=0
while [ "$i" -lt 60 ]; do
	code=$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://127.0.0.1:$port/healthz" || true)
	if [ "$code" = 200 ]; then ready=1; break; fi
	i=$((i + 1))
	printf .
	sleep 1
done
echo
[ "$ready" = 1 ] || fail "healthz never answered 200 (last: ${code:-none})"

# A fresh install must NOT serve the library to anyone who finds the port. It asks for the first
# account instead, and holds everything else behind that page.
location=$(curl -s -o /dev/null -m 10 -w '%{redirect_url}' "http://127.0.0.1:$port/")
case "$location" in
	*"/setup") ;;
	*) fail "root went to '$location', want /setup on a server with no account" ;;
esac
code=$(curl -s -o /dev/null -m 10 -w '%{http_code}' "http://127.0.0.1:$port/api/libraries")
[ "$code" = 303 ] || fail "library list returned $code before setup, want a redirect"

# The first-run page itself must work, and creating the account must open the library.
code=$(curl -s -o /tmp/install-smoke-setup.html -m 10 -w '%{http_code}' "http://127.0.0.1:$port/setup")
[ "$code" = 200 ] || fail "setup page returned $code, want 200"
grep -q 'name="password"' /tmp/install-smoke-setup.html || fail "setup page has no password field"
token=$(sed -n 's/.*name="csrf" value="\([^"]*\)".*/\1/p' /tmp/install-smoke-setup.html | head -1)
[ -n "$token" ] || fail "setup page has no csrf token"
rm -f /tmp/install-smoke-setup.html
code=$(curl -s -o /dev/null -m 20 -w '%{http_code}' -X POST "http://127.0.0.1:$port/setup" \
	--data-urlencode "csrf=$token" --data-urlencode "username=owner" \
	--data-urlencode "password=a long enough password" --data-urlencode "confirm=a long enough password")
[ "$code" = 303 ] || fail "creating the first account returned $code, want 303"

# With an account in place the door is the ordinary sign-in page, and the setup page is closed.
location=$(curl -s -o /dev/null -m 10 -w '%{redirect_url}' "http://127.0.0.1:$port/")
case "$location" in
	*"/login"*) ;;
	*) fail "root went to '$location' after setup, want the sign-in page" ;;
esac

# An empty library is a normal first-run state, not a crash: the container keeps running.
[ "$(docker inspect -f '{{.State.Running}}' "$name")" = true ] || fail "container is not running"

echo "== restarting on the same volumes"
docker restart "$name" >/dev/null || fail "restart failed"
ready=0
i=0
while [ "$i" -lt 60 ]; do
	code=$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://127.0.0.1:$port/healthz" || true)
	if [ "$code" = 200 ]; then ready=1; break; fi
	i=$((i + 1))
	sleep 1
done
[ "$ready" = 1 ] || fail "healthz did not come back after a restart"

# The account is on the volume, so a restart must not reopen the setup page or drop the gate.
location=$(curl -s -o /dev/null -m 10 -w '%{redirect_url}' "http://127.0.0.1:$port/")
case "$location" in
	*"/login"*) ;;
	*) fail "after a restart root went to '$location', want the sign-in page" ;;
esac


# A backup nobody has restored is a hope, not a backup. Copy the config volume the way a person would
# (docker cp or a tar of the folder), boot a *second*, untouched container on it, and check that the
# account and the closed door came with it.
echo "== restoring the config volume into a fresh container"
restored="${name}-restored"
docker volume create "${name}-restore" >/dev/null
docker run --rm -v "$name-config:/from:ro" -v "${name}-restore:/to" alpine:3.22 \
	sh -c 'cp -a /from/. /to/' || fail "copying the config volume failed"
docker run -d --name "$restored" -p "127.0.0.1:$((port + 1)):8080" \
	-v "${name}-restore:/config" -v "$name-data:/data" -v "$name-books:/books" \
	"$image" >/dev/null || fail "the restored container did not start"
restoredPort=$((port + 1))
ready=0
i=0
while [ "$i" -lt 60 ]; do
	code=$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://127.0.0.1:$restoredPort/healthz" || true)
	if [ "$code" = 200 ]; then ready=1; break; fi
	i=$((i + 1))
	sleep 1
done
[ "$ready" = 1 ] || fail "the restored container never became healthy"
location=$(curl -s -o /dev/null -m 10 -w '%{redirect_url}' "http://127.0.0.1:$restoredPort/")
case "$location" in
	*"/login"*) ;;
	*) fail "restored install went to '$location', want the sign-in page" ;;
esac
# And the account itself survived: sign in with the credentials created earlier.
jar=$(mktemp)
csrf=$(curl -s -c "$jar" "http://127.0.0.1:$restoredPort/login" | sed -n 's/.*name="csrf" value="\([^"]*\)".*/\1/p' | head -1)
[ -n "$csrf" ] || fail "restored install served no sign-in form"
code=$(curl -s -b "$jar" -c "$jar" -o /dev/null -m 20 -w '%{http_code}' -X POST "http://127.0.0.1:$restoredPort/login" \
	--data-urlencode "csrf=$csrf" --data-urlencode "username=owner" --data-urlencode "password=a long enough password")
rm -f "$jar"
[ "$code" = 303 ] || fail "the restored account could not sign in ($code)"
docker rm -f "$restored" >/dev/null 2>&1 || true
docker volume rm -f "${name}-restore" >/dev/null 2>&1 || true

echo "PASS: a fresh install asks for its first account, keeps it across a restart and a restore, and never serves the library unauthenticated"
