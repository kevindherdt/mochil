#!/usr/bin/env bash
# Publish the public mirror of this repository.
#
# export-public.sh deliberately refuses to write into an existing path, which is right -- it protects
# against half-overwriting a tree that is already published. Doing that by hand, though, meant moving
# the previous export aside on every release, and those copies pile up: one afternoon left fifty-odd
# `mochil-prev*` directories and 2.7 GB in /mnt/data/DEV.
#
# So: export into a scratch directory outside the destination, then swap it into place. Exactly one
# export exists at any moment, and a failed export leaves the published tree untouched.
#
#   ./scripts/publish-public.sh                 # export, commit, push main
#   ./scripts/publish-public.sh --tag 0.1.36    # ... and tag v0.1.36
#
# The APK is not handled here: signing needs the gitignored keystore, so that upload stays manual and
# has to happen after the Release workflow has created the release object. See AGENTS.md.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
dest=${MOCHIL_PUBLIC_DIR:-/mnt/data/DEV/mochil}
parent=$(dirname "$dest")
name=$(basename "$dest")
scratch="$parent/.$name-export-$$"
tag=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --tag) tag=${2:?--tag needs a version}; shift 2 ;;
    *) echo "usage: $(basename "$0") [--tag <version>]" >&2; exit 2 ;;
  esac
done

cleanup() { rm -rf "$scratch"; }
trap cleanup EXIT

echo "exporting to a scratch directory..."
rm -rf "$scratch"
"$root/scripts/export-public.sh" --gates --git "$scratch" >/dev/null

# The published tree is only replaced once the new one is complete and has passed the gates.
rm -rf "$dest"
mv "$scratch" "$dest"
trap - EXIT
echo "published tree replaced: $dest"

cd "$dest"
git remote add origin git@github.com:kevindherdt/mochil.git 2>/dev/null || true
git config core.sshCommand "ssh -i ${MOCHIL_SSH_KEY:-/root/.ssh/hermes_dev}"
git config user.name "Kevin De Herdt"
git config user.email "kevin@kdhmedia.net"
git push --force origin main 2>&1 | tail -1

if [[ -n $tag ]]; then
  git tag -a "v$tag" -m "Mochil $tag"
  git push --force origin "v$tag" 2>&1 | tail -1
  echo "tagged v$tag; attach the APK once the Release workflow finishes"
fi
