#!/usr/bin/env bash
# Publish the public mirror of this repository.
#
# export-public.sh deliberately refuses to write into an existing path, which is right -- it protects
# against half-overwriting a tree that is already published. Doing that by hand, though, meant moving
# the previous export aside on every release, and those copies pile up: one afternoon left fifty-odd
# `mochil-prev*` directories and 2.7 GB in /mnt/data/DEV.
#
# So: export into a scratch directory outside the destination, then lay that tree onto the published
# repository and commit it normally. The history this creates is the *public* repository's own -- the
# development tree is still never pushed -- and a release now has a real parent, so GitHub shows the
# files that changed instead of listing every file in the tree on every release.
#
#   ./scripts/publish-public.sh                 # export, commit, push main
#   ./scripts/publish-public.sh --tag 0.1.36    # ... and tag v0.1.36
#
# The APK is not handled here: signing needs the gitignored keystore, so that upload stays manual and
# has to happen after the Release workflow has created the release object. See AGENTS.md.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
dest=${MOCHIL_PUBLIC_DIR:-/mnt/data/DEV/mochil}
remote=${MOCHIL_PUBLIC_REMOTE:-git@github.com:kevindherdt/mochil.git}
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
"$root/scripts/export-public.sh" --gates "$scratch" >/dev/null

# The published repository keeps its history. Take it into place before the new tree so the commit
# below has a parent; an installation that has never published starts one instead.
if [[ ! -d $dest/.git ]]; then
  rm -rf "$dest"
  mkdir -p "$parent"
  if GIT_SSH_COMMAND="ssh -i ${MOCHIL_SSH_KEY:-/root/.ssh/hermes_dev}" \
       git ls-remote --heads "$remote" main >/dev/null 2>&1; then
    echo "cloning the published history from $remote"
    GIT_SSH_COMMAND="ssh -i ${MOCHIL_SSH_KEY:-/root/.ssh/hermes_dev}" \
      git clone --quiet --branch main "$remote" "$dest"
  else
    echo "no published history yet: starting one"
    mkdir -p "$dest"
    git -C "$dest" init -q -b main
  fi
fi

cd "$dest"
git remote get-url origin >/dev/null 2>&1 || git remote add origin "$remote"
git config core.sshCommand "ssh -i ${MOCHIL_SSH_KEY:-/root/.ssh/hermes_dev}"
git config user.name "Kevin De Herdt"
git config user.email "kevin@kdhmedia.net"

# Everything except .git is replaced by the export, deletions included. That is what makes each commit
# describe one release instead of restating the whole tree.
rsync -a --delete --exclude '.git/' "$scratch/" "$dest/"
trap - EXIT
rm -rf "$scratch"

version=$(cat VERSION 2>/dev/null || echo "unversioned")
git add -A
if git diff --cached --quiet; then
  echo "the published tree already matches this export; nothing to commit"
else
  changed=$(git diff --cached --name-only | wc -l | tr -d ' ')
  summary=$(git diff --cached --shortstat | sed 's/^ *//')
  git commit -q -m "Mochil $version

$changed files changed: $summary

Exported from the development tree; CHANGELOG.md says what changed for readers."
  echo "committed $changed files on top of $(git rev-parse --short 'HEAD^' 2>/dev/null || echo 'the first commit')"
fi

git push origin main 2>&1 | tail -1

if [[ -n $tag ]]; then
  if git rev-parse -q --verify "refs/tags/v$tag" >/dev/null; then
    echo "tag v$tag already exists; not moved"
  else
    git tag -a "v$tag" -m "Mochil $tag"
    git push origin "v$tag" 2>&1 | tail -1
    echo "tagged v$tag; the release workflow builds and attaches the APK itself"
  fi
fi
