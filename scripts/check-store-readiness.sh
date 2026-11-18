#!/usr/bin/env bash
set -euo pipefail

readonly script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
readonly repository_root=$(cd -- "$script_dir/.." && pwd -P)
cd -- "$repository_root"

# Executable manga-extension integrations were retired. Keep the active clients,
# browser surface, dependencies, and optional packaged artifacts free of their
# runtime identities. Legacy config decoding accepts only the old generic
# `external_sources` field so installations can start and discard it without
# retaining a removed provider or executable runtime identity.
readonly forbidden='suwayomi|tachidesk|tachiyomi|mihon|keiyoushi'
readonly source_forbidden="$forbidden|DexClassLoader|PathClassLoader|proguard-extension-rules"
readonly dependency_forbidden="$source_forbidden|jitpack\.io"
readonly store_updater_forbidden='bobobooks-update\.apk|application/vnd\.android\.package-archive|Downloading update|Download and install|Install unknown apps'
readonly store_permission_forbidden='android\.permission\.(REQUEST_INSTALL_PACKAGES|QUERY_ALL_PACKAGES|MANAGE_EXTERNAL_STORAGE|READ_EXTERNAL_STORAGE|WRITE_EXTERNAL_STORAGE)'
readonly deployment_identity_forbidden='([[:alnum:]-]+\.)*kdhmedia\.net'
readonly main_manifest='android/app/src/main/AndroidManifest.xml'
readonly source_roots=(
  android/app/src/main
  android/app/src/direct
  android/app/src/store
  cmd
  internal
  internal/web/static
)
readonly dependency_files=(
  android/app/build.gradle.kts
  android/app/proguard-rules.pro
  android/settings.gradle.kts
  go.mod
  go.sum
)
readonly android_settings='android/settings.gradle.kts'
readonly dockerignore='.dockerignore'
for private_exclusion in '.env' '.env.*' 'config' 'media' 'compose.override.yaml' 'android/local.properties'; do
  if ! grep -qxF "$private_exclusion" "$dockerignore"; then
    echo "Docker build context must exclude local runtime data: $private_exclusion" >&2
    exit 1
  fi
done
readonly gitignore='.gitignore'

failed=0
for signing_exclusion in 'android/.signing' 'android/keystore.properties' '**/*.keystore' '**/*.jks'; do
  if ! grep -qxF "$signing_exclusion" "$dockerignore"; then
    echo "Docker build context must exclude signing material: $signing_exclusion" >&2
    failed=1
  fi
done
declare -a packaged_artifacts=("$@")

# A disconnected experimental module is still build-system and supply-chain
# surface. Keep the distributable Android project intentionally single-module;
# generated build directories are irrelevant and are checked separately only
# when an APK is explicitly supplied.
mapfile -t included_android_projects < <(sed -nE 's/^[[:space:]]*include\("([^"]+)"\)[[:space:]]*$/\1/p' "$android_settings")
if (( ${#included_android_projects[@]} != 1 )) || [[ "${included_android_projects[0]:-}" != ':app' ]]; then
  echo "Android settings must include only the distributable :app module" >&2
  printf 'Found module: %s\n' "${included_android_projects[@]:-none}" >&2
  failed=1
fi
for retired_module in android/extension-api android/extensions android/source-engine; do
  if [[ -e "$retired_module" ]]; then
    echo "Retired Android integration module still exists: $retired_module" >&2
    failed=1
  fi
done

# The generic legacy field may exist only at the strict config decoder and the
# settings redaction boundary. Tests may name it, but distributable runtime code
# must never acquire a third owner or turn the migration shim back into a feature.
mapfile -t legacy_source_owners < <(
  grep -RIl --exclude='*_test.go' --exclude-dir=vendor 'external_sources' \
    android/app/src/main android/app/src/direct android/app/src/store cmd internal 2>/dev/null || true
)
for owner in "${legacy_source_owners[@]}"; do
  case "$owner" in
    internal/appconfig/config.go|internal/settings/service.go) ;;
    *)
      echo "Retired external_sources field gained a runtime owner: $owner" >&2
      failed=1
      ;;
  esac
done
if ! grep -qF 'cfg.RetiredExternalSources = nil' internal/appconfig/config.go || \
   [[ $(grep -cF 'delete(document, "external_sources")' internal/settings/service.go) -ne 1 ]] || \
   [[ $(grep -cF 'delete(incoming, "external_sources")' internal/settings/service.go) -ne 1 ]]; then
  echo "Legacy external_sources compatibility must remain discard-only" >&2
  failed=1
fi

if ! grep -qxF 'android/**/build' "$dockerignore" || ! grep -qxF 'android/.kotlin' "$dockerignore"; then
  echo "Docker build context must exclude every Android build output and Kotlin workspace cache" >&2
  failed=1
fi
if ! grep -qxF '/android/**/build/' "$gitignore" || ! grep -qxF '/android/.kotlin/' "$gitignore"; then
  echo "Repository ignores must exclude every Android build output and Kotlin workspace cache" >&2
  failed=1
fi

require_manifest_value() {
  local pattern=$1 description=$2
  if ! grep -qE "$pattern" "$main_manifest"; then
    echo "Android manifest must keep $description" >&2
    failed=1
  fi
}

require_manifest_value 'android:allowBackup="false"' 'application backup disabled so tokens and private state are not exported'
require_manifest_value 'android:usesCleartextTraffic="false"' 'cleartext application traffic disabled'
if ! grep -A4 'androidx\.core\.content\.FileProvider' "$main_manifest" | grep -q 'android:exported="false"'; then
  echo "Android FileProvider must remain non-exported" >&2
  failed=1
fi

readonly shared_updater='android/app/src/main/java/net/kdhmedia/mochil/update/AppUpdateViewModel.kt'
readonly direct_updater='android/app/src/direct/java/net/kdhmedia/mochil/update/AppUpdateViewModel.kt'
readonly store_updater='android/app/src/store/java/net/kdhmedia/mochil/update/AppUpdateViewModel.kt'
if [[ -e "$shared_updater" || ! -f "$direct_updater" || ! -f "$store_updater" ]]; then
  echo "The installer-capable updater must live only in the direct flavor, with an inert store implementation" >&2
  failed=1
fi
if matches=$(grep -RIniE "$store_updater_forbidden" android/app/src/store 2>/dev/null); then
  echo "Store flavor source contains direct-install updater behavior:" >&2
  echo "$matches" >&2
  failed=1
fi

if (( ${#packaged_artifacts[@]} == 0 )); then
  while IFS= read -r -d '' artifact; do
    packaged_artifacts+=("$artifact")
  done < <(find android/app/build/outputs -type f \( -path '*/optimized/*.apk' -o -path '*/storeOptimized/*.aab' \) -print0 2>/dev/null)
fi

find_aapt2() {
  if command -v aapt2 >/dev/null 2>&1; then
    command -v aapt2
    return
  fi
  # CI images ship the SDK with build-tools entries that may be symlinks, and the first root that
  # exists is not necessarily the one with aapt2 in it, so search every candidate and accept links.
  local sdk_root found
  for sdk_root in "${ANDROID_SDK_ROOT:-}" "${ANDROID_HOME:-}" /opt/android-sdk /opt/android /usr/local/lib/android/sdk "$HOME/Android/Sdk"; do
    [[ -n "$sdk_root" && -d "$sdk_root" ]] || continue
    found=$(find "$sdk_root/build-tools" -mindepth 2 -maxdepth 2 \( -type f -o -type l \) -name aapt2 2>/dev/null | sort -V | tail -n 1)
    if [[ -n "$found" ]]; then
      printf '%s\n' "$found"
      return
    fi
  done
  return 1
}

is_store_artifact() {
  [[ "$(basename "$1")" == *store* || "$1" == */store/* || "$1" == */storeOptimized/* ]]
}
for root in "${source_roots[@]}"; do
  if matches=$(grep -RIniE --exclude-dir=vendor "$source_forbidden" "$root" 2>/dev/null); then
    echo "Retired extension integration found in active source:" >&2
    echo "$matches" >&2
    failed=1
  fi
  if matches=$(grep -RIniE --exclude-dir=vendor "$deployment_identity_forbidden" "$root" 2>/dev/null); then
    echo "Personal deployment hostname found in distributable source:" >&2
    echo "$matches" >&2
    failed=1
  fi
done

for file in "${dependency_files[@]}"; do
  if matches=$(grep -IniE "$dependency_forbidden" "$file" 2>/dev/null); then
    echo "Retired extension integration found in dependencies: $file" >&2
    echo "$matches" >&2
    failed=1
  fi
done

for artifact in "${packaged_artifacts[@]}"; do
  [[ -f "$artifact" ]] || { echo "Package artifact does not exist: $artifact" >&2; failed=1; continue; }
  case "$artifact" in
    *.apk|*.aab) ;;
    *) echo "Unsupported package artifact: $artifact" >&2; failed=1; continue ;;
  esac
  # Absence of forbidden strings is not evidence that an unreadable package is
  # safe. Validate the archive before pipelines can mask an unzip failure.
  if ! unzip -tqq "$artifact" >/dev/null 2>&1; then
    echo "Package artifact is corrupt or unreadable: $artifact" >&2
    failed=1
    continue
  fi
  if matches=$({ unzip -Z1 "$artifact"; unzip -p "$artifact"; } 2>/dev/null | strings | grep -iE "$forbidden"); then
    echo "Retired extension integration found in packaged artifact: $artifact" >&2
    echo "$matches" >&2
    failed=1
  fi
  if matches=$(unzip -p "$artifact" 2>/dev/null | strings | grep -iE "$deployment_identity_forbidden"); then
    echo "Personal deployment hostname found in packaged artifact: $artifact" >&2
    echo "$matches" >&2
    failed=1
  fi
  if is_store_artifact "$artifact"; then
    if matches=$(unzip -p "$artifact" 2>/dev/null | strings | grep -iE "$store_updater_forbidden"); then
      echo "Store artifact contains the direct-install updater: $artifact" >&2
      echo "$matches" >&2
      failed=1
    fi
  fi
  if [[ "$artifact" == *.apk ]] && is_store_artifact "$artifact"; then
    if ! aapt2_bin=$(find_aapt2) || [[ -z "$aapt2_bin" ]]; then
      echo "Cannot inspect store APK permissions because aapt2 is unavailable (ANDROID_SDK_ROOT=${ANDROID_SDK_ROOT:-unset} ANDROID_HOME=${ANDROID_HOME:-unset} PATH=${PATH})" >&2
      failed=1
    else
      permission_dump=$("$aapt2_bin" dump permissions "$artifact")
      if matches=$(grep -E "$store_permission_forbidden" <<<"$permission_dump"); then
        echo "Store APK requests a forbidden broad-storage or package-discovery permission: $artifact" >&2
        echo "$matches" >&2
        failed=1
      fi
    fi
  fi
done

exit "$failed"
