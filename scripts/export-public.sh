#!/usr/bin/env bash
# Builds the public repository from this development tree.
#
# It copies a curated list of files into a fresh directory, leaves the git history behind, and
# refuses to finish if it finds anything that looks like a credential or a home-network address.
# The dev tree is never modified.
#
#   ./scripts/export-public.sh /mnt/data/DEV/mochil          # build the export
#   ./scripts/export-public.sh --git /mnt/data/DEV/mochil    # build it and make the first commit
#
# The output directory is deliberately outside this repository: two git repositories in one tree
# is how people end up committing the wrong thing.
set -euo pipefail

src=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
do_git=0
do_gates=0
while [[ ${1:-} == --* ]]; do
  case $1 in
    --git) do_git=1 ;;
    --gates) do_gates=1 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done
dest=${1:-}
if [[ -z $dest ]]; then
  echo "usage: $(basename "$0") [--git] [--gates] <destination>" >&2
  exit 2
fi

# Everything the public repo needs. Anything not named here does not ship, which is the point:
# a deny-by-default list cannot leak a file someone added later.
# Note: docs/ is deliberately not shipped wholesale. It holds 148 dated internal audits and
# per-release notes from before the rename; the three files below are the ones a reader needs.
dirs=(
  .github cmd internal web design
  android/app android/gradle android/gradle.properties android/gradlew android/gradlew.bat
  android/build.gradle.kts android/settings.gradle.kts android/lint.xml android/gradle.properties
  config contracts deployment example nix scripts screenshots
)
root_files=(
  .gitignore .dockerignore .graphifyignore .env.example CHANGELOG.md VERSION CONTRIBUTING.md Containerfile Containerfile.calibre Containerfile.narration
  DESIGN.md LICENSE PRODUCT.md README.md SECURITY.md THIRD_PARTY_NOTICES.md TRADEMARK.md
  compose.calibre.yaml compose.narration.yaml compose.yaml
  go.mod go.sum package.json package-lock.json mochil.example.json flake.nix android/README.md
  docs/ARCHITECTURE_ROADMAP.md docs/NARRATION.md
)

excludes=(
  --exclude='.DS_Store' --exclude='*.pyc' --exclude='__pycache__/'
  --exclude='build/' --exclude='.gradle/' --exclude='.kotlin/'
  --exclude='local.properties' --exclude='keystore.properties' --exclude='.signing/'
  --exclude='*.apk' --exclude='*.aab' --exclude='*.jks' --exclude='*.keystore'
  --exclude='*.bak*' --exclude='*.orig' --exclude='*.rej'
  --exclude='dist/' --exclude='node_modules/' --exclude='HERMES_PLAN.md' --exclude='PERFORMANCE.md'
)

if [[ -e $dest ]]; then
  echo "refusing to write into an existing path: $dest" >&2
  echo "move it aside first, or choose another destination" >&2
  exit 1
fi
mkdir -p "$dest"

copied=0
for d in "${dirs[@]}"; do
  [[ -e $src/$d ]] || continue
  mkdir -p "$dest/$(dirname "$d")"
  rsync -a "${excludes[@]}" "$src/$d" "$dest/$(dirname "$d")/"
  copied=$((copied + 1))
done
for f in "${root_files[@]}"; do
  [[ -f $src/$f ]] || continue
  mkdir -p "$dest/$(dirname "$f")"
  cp -p "$src/$f" "$dest/$f"
  copied=$((copied + 1))
done

# config/ ships as an empty folder: the server writes its own config on first run, and the dev
# copy here holds live provider keys.
rm -rf "$dest/config"
mkdir -p "$dest/config"
: > "$dest/config/.gitkeep"

echo "copied $copied paths into $dest"

# The allowlist above is deny-by-default, which is what keeps a secret out — but it silently drops
# files added later, and that produced three separate CI failures (a gate script's .dockerignore,
# the android app module's build files, and docs the README links to). So the allowlist has to
# agree with the tree: every tracked file either ships or is named as private here.
private_paths=(
  'hindsight/' 'artifacts/' '.ornith-proposals/' '.qwen-proposals/' '.claude/' '.impeccable/'
  'android/.signing/' 'android/local.properties' 'android/.maestro/' 'keystore.properties'
  'config/' 'docs/' 'tools/' 'notes/' 'media/' 'packaging/' 'references/'
  'dogfood-output-2026-11-07/' 'chatgpt-audit.md' 'android-perf-audit.md' 'AGENTS.md'
  'HERMES_PLAN.md' 'PERFORMANCE.md'
)
missing=()
while IFS= read -r f; do
  skip=0
  for p in "${private_paths[@]}"; do [[ $f == "$p"* ]] && { skip=1; break; }; done
  (( skip )) && continue
  [[ -e $dest/$f ]] || missing+=("$f")
done < <(git -C "$src" ls-files)
if (( ${#missing[@]} )); then
  echo "STOP: these tracked files are neither shipped nor listed as private:" >&2
  printf '  %s\n' "${missing[@]}" >&2
  echo "add them to dirs/root_files, or to private_paths if they must not ship. Nothing was committed." >&2
  exit 1
fi
echo "complete: every tracked file ships, except the ones listed as private"

# Nothing below is allowed to reach a public repository. The scan runs on the copy, so what it
# inspects is exactly what would be published.
echo
echo "scanning the export for secrets and home-network details..."
# Two rules, because they mean different things. Credentials are wrong anywhere. A private-range
# address is a leak in shipped code and documentation, but it is exactly right inside a test that
# asserts private ranges are refused — so those files are exempt from the address rule only.
creds=$(cd "$dest" && grep -rIlE \
  'oc_sk_[A-Za-z0-9_-]{8,}|sk-[A-Za-z0-9]{16,}|ghp_[A-Za-z0-9]{20,}|gho_[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{12,}|-----BEGIN [A-Z ]*PRIVATE KEY|x-opencode-session' \
  --exclude='export-public.sh' . 2>/dev/null || true)

addresses=$(cd "$dest" && grep -rIlE \
  '192\.168\.[0-9]+\.[0-9]+|\b10\.[0-9]+\.[0-9]+\.[0-9]+\b|\b172\.(1[6-9]|2[0-9]|3[01])\.[0-9]+\.[0-9]+\b' \
  --exclude='export-public.sh' --exclude='*_test.go' --exclude='*Test.kt' --exclude='*.test.mjs' \
  --exclude-dir=test . 2>/dev/null || true)

hits=$(printf '%s\n%s\n' "$creds" "$addresses" | sed '/^$/d' | sort -u)

problems=0
if [[ -n $hits ]]; then
  echo "found credential or home-network patterns in:" >&2
  printf '  %s\n' $hits >&2
  problems=1
fi

envfiles=$(cd "$dest" && find . -type f \( -name '.env' -o -name '*.env' -o -name '.env.*' \) ! -name '*.example' -print)
if [[ -n $envfiles ]]; then
  echo "found environment files that are not templates:" >&2
  printf '  %s\n' $envfiles >&2
  problems=1
fi

binaries=$(cd "$dest" && find . -type f \( -name '*.apk' -o -name '*.aab' -o -name '*.jks' -o -name '*.keystore' -o -name '*.pyc' \) -print)
if [[ -n $binaries ]]; then
  echo "found build output or signing material:" >&2
  printf '  %s\n' $binaries >&2
  problems=1
fi

if (( problems )); then
  echo
  echo "STOP: the export is not clean. Fix the findings above (or add an exclude) and run again." >&2
  echo "Nothing was committed. The copy is at $dest for inspection." >&2
  exit 1
fi
echo "clean: no credentials, environment files, signing material or LAN addresses."

files=$(cd "$dest" && find . -type f | wc -l | tr -d ' ')
bytes=$(du -sh "$dest" | cut -f1)
echo
echo "export: $files files, $bytes"

if (( do_gates )); then
  echo
  echo "running the export's own gates (this is what CI will do)..."
  ( cd "$dest" && npm ci --silent >/dev/null 2>&1 && npm run test:frontend >/tmp/gates-frontend.log 2>&1 ) \
    || { echo "STOP: frontend tests failed in the export:" >&2; tail -20 /tmp/gates-frontend.log >&2; exit 1; }
  ( cd "$dest" && ./scripts/check-store-readiness.sh ) \
    || { echo "STOP: check-store-readiness.sh failed in the export" >&2; exit 1; }
  ( cd "$dest" && ./scripts/check-module-sizes.sh ) \
    || { echo "STOP: check-module-sizes.sh failed in the export" >&2; exit 1; }
  echo "gates passed in the export"
  rm -rf "$dest/node_modules"
fi

if (( do_git )); then
  cd "$dest"
  git init -q -b main
  git add -A
  git -c user.name="Kevin De Herdt" -c user.email="kevin@kdhmedia.net" \
    commit -q -m "Mochil: a self-hosted library for books, comics, audio and video"
  echo "first commit: $(git log --oneline -1)"
  echo "add the remote when you are ready: git remote add origin git@github.com:kevindherdt/mochil.git"
else
  echo "inspect it, then re-run with --git to create the repository."
fi
