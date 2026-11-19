#!/usr/bin/env bash
# Release notes are the app's own changelog, not a list of commit subjects: a release should tell
# someone who runs Mochil what changed for them, in the words already written in
# internal/changelog/changelog.json. Falls back to the commit list when a version has no entry.
set -euo pipefail

version="${1:-}"
if [ -z "$version" ]; then
  version="$(tr -d '[:space:]' < "$(dirname "$0")/../VERSION")"
fi

repo="${GITHUB_REPOSITORY:-kevindherdt/mochil}"
changelog="$(dirname "$0")/../internal/changelog/changelog.json"

notes="$(python3 - "$changelog" "$version" <<'PY'
import json, sys

path, version = sys.argv[1], sys.argv[2].lstrip("v")
entries = json.load(open(path))["entries"]
mine = [e for e in entries if str(e.get("version")) == version]

def lines(items):
    return "\n".join(f"- {item}" for item in items)

out = []
new = [i for e in mine for i in (e.get("new") or [])]
changes = [i for e in mine for i in (e.get("changes") or [])]
if new:
    out.append("### New\n\n" + lines(new))
if changes:
    out.append("### Changes\n\n" + lines(changes))
print("\n\n".join(out))
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

**Full changelog:** https://github.com/${repo}/compare/v${version}...main
EOF
