#!/usr/bin/env bash
# Release notes are the app's own changelog, not a list of commit subjects: a release should tell
# someone who runs Mochil what changed for them, in the words already written in
# internal/changelog/changelog.json.
#
# A held-back release is the normal case here — several versions get written before one is tagged — so the
# notes cover every entry since the last published release, not only the version being tagged. Otherwise the
# notes for 0.1.58 would be silent about the work that shipped as 0.1.56 and 0.1.57.
set -euo pipefail

version="${1:-}"
if [ -z "$version" ]; then
  version="$(tr -d '[:space:]' < "$(dirname "$0")/../VERSION")"
fi
version="${version#v}"

repo="${GITHUB_REPOSITORY:-kevindherdt/mochil}"
changelog="$(dirname "$0")/../internal/changelog/changelog.json"

# The last release people could actually install. Empty when this is the first, or when the API is not
# reachable from here — the script then falls back to the single version's own notes.
previous=""
if command -v gh >/dev/null 2>&1; then
  previous="$( { gh release list --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null || true; } | tr -d '[:space:]' | sed 's/^v//')"
fi

# A rehearsal needs to say what the last release was without asking GitHub.
previous="${MOCHIL_PREVIOUS_RELEASE:-$previous}"

notes="$(python3 - "$changelog" "$version" "$previous" <<'PY'
import json, sys

path, version, previous = sys.argv[1], sys.argv[2], sys.argv[3]


def parts(text):
    try:
        return tuple(int(piece) for piece in str(text).split("."))
    except ValueError:
        return None


target, floor = parts(version), parts(previous) if previous else None
entries = json.load(open(path))["entries"]

mine = [e for e in entries if str(e.get("version")) == version]
if not mine:
    sys.exit(0)

# Everything newer than the last published release, newest first: the held-back versions plus this one.
if floor:
    included = []
    for entry in entries:
        number = parts(entry.get("version"))
        if number is None or number > target:
            continue
        if number > floor:
            included.append(entry)
    if not any(str(e.get("version")) == version for e in included):
        included = mine
else:
    included = mine


def lines(items):
    return "\n".join(f"- {item}" for item in items)


sections = []
for entry in included:
    new = entry.get("new") or []
    changes = entry.get("changes") or []
    fixed = entry.get("fixed") or []
    performance = entry.get("performance") or []
    body = []
    if new:
        body.append("**New**\n\n" + lines(new))
    if changes:
        body.append(lines(changes))
    if fixed:
        body.append("**Fixed**\n\n" + lines(fixed))
    if performance:
        body.append("**Performance**\n\n" + lines(performance))
    if not body:
        continue
    heading = f"### {entry.get('version')}" if len(included) > 1 else ""
    sections.append((heading + "\n\n" if heading else "") + "\n\n".join(body))

print("\n\n".join(sections))
PY
)"

if [ -z "$notes" ]; then
  notes="No changelog entry for ${version}."
fi

cat <<EOF
${notes}

### Install

\`\`\`
docker pull ghcr.io/${repo}:${version}
\`\`\`

Or with the compose file from the repository:

\`\`\`
docker compose pull && docker compose up -d
\`\`\`

The container serves http://localhost:8099 by default, and the first visit asks you to create the
account. Your libraries and settings live in \`config/\`; \`data/catalogs/\` holds sign-ins, pairing
and reading state. Back up both, and the rest of \`data/\` can be deleted and rebuilt.

### Upgrading

\`\`\`
docker compose pull && docker compose up -d
\`\`\`

Settings are read from the mounted \`config/\`, so they survive the upgrade.

**Full changelog:** https://github.com/${repo}/blob/main/internal/changelog/changelog.json
EOF
