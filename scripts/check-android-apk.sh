#!/usr/bin/env bash
# Refuse to serve an APK that is not this release.
#
# The server's download screen and the app's own "is there a newer version?" check both answer from
# VERSION, while the file people install is whatever compose mounts at /downloads/mochil-android.apk.
# That mount pointed at a 0.1.16 build for dozens of releases: the app said 0.1.20 (it was labelled with
# a hard-coded number), the server advertised 0.1.62, and a phone that downloaded it still had the Paper
# theme because the file predated its removal. A missing or stale APK here is invisible until someone
# installs it, so it is checked before a deploy instead of after a complaint.
#
#   ./scripts/check-android-apk.sh                  # check the file compose mounts
#   ./scripts/check-android-apk.sh path/to/app.apk  # check a specific build
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
want=$(tr -d '[:space:]' < "$root/VERSION")

apk=${1:-}
if [[ -z $apk ]]; then
  apk=$(sed -n 's#.*- \(\./artifacts/releases/[^:]*\):/downloads/mochil-android.apk.*#\1#p' "$root/compose.override.yaml" | head -1)
fi
[[ -n $apk ]] || { echo "compose.override.yaml does not mount an APK at /downloads/mochil-android.apk" >&2; exit 1; }
[[ $apk = /* ]] || apk="$root/${apk#./}"
[[ -f $apk ]] || { echo "the mounted APK is missing: $apk" >&2; exit 1; }

sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/android-sdk}}
aapt=$(ls "$sdk"/build-tools/*/aapt2 2>/dev/null | sort -V | tail -1 || true)
[[ -n $aapt ]] || { echo "aapt2 not found under $sdk/build-tools" >&2; exit 1; }

# sed -n 1p rather than head: under pipefail a closed pipe reports 141 and reads as a broken APK.
line=$("$aapt" dump badging "$apk" 2>/dev/null | sed -n 1p)
name=$(sed -n "s/.*versionName='\([^']*\)'.*/\1/p" <<<"$line")
code=$(sed -n "s/.*versionCode='\([0-9]*\)'.*/\1/p" <<<"$line")
echo "mounted: ${apk#"$root"/}  ${name:-<none>} (${code:-none})   release: $want"

[[ $name == "$want" ]] || {
  echo "the APK served to phones is $name, but this release is $want" >&2
  exit 1
}
