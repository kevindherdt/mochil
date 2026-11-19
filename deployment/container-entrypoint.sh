#!/bin/sh
set -eu

# Docker creates a missing bind-mount directory as root. Prepare only the two
# application-owned roots, then permanently drop privileges before Mochil
# reads configuration or starts serving requests.
#
# Mochil runs as an unprivileged user. Set MOCHIL_UID and MOCHIL_GID to the
# owner of your notes folder when the Wiki must edit it, so pages it writes stay
# editable from Obsidian and file sync.
uid="${MOCHIL_UID:-65532}"
gid="${MOCHIL_GID:-$uid}"
case "$uid$gid" in
  *[!0-9]*|"") echo "MOCHIL_UID and MOCHIL_GID must be numbers" >&2; exit 2 ;;
esac
if [ "$uid" = 0 ] || [ "$gid" = 0 ]; then
  echo "MOCHIL_UID and MOCHIL_GID must not be root (0)" >&2; exit 2
fi
# These two folders hold only Mochil' own files. When the user it runs as
# changes, hand everything in them over so its state stays readable. Look at the
# owner before creating or re-owning anything.
for owned in /config /data; do
  if [ -d "$owned" ] && [ "$(stat -c %u "$owned")" != "$uid" ]; then
    chown -R "$uid:$gid" "$owned"
  fi
done
install -d -m 0750 -o "$uid" -g "$gid" /config /data

# Restores and host-side editors can replace this private file with a new
# owner. Repair only the conventional application config, never media trees
# or a symlink target outside the config mount. Preserve its private mode.
if [ -f /config/config.json ] && [ ! -L /config/config.json ]; then
  chown "$uid:$gid" /config/config.json
fi

if command -v su-exec >/dev/null 2>&1; then
  exec su-exec "$uid:$gid" /mochil "$@"
fi
exec gosu "$uid:$gid" /mochil "$@"
