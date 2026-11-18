# Releasing Mochil

One number, in one file: [`VERSION`](../VERSION). Everything else is derived from it.

## The scheme

`MAJOR.MINOR.PATCH`, starting at **0.1.0**. While the major number is 0, a minor bump may change
configuration or the API; after 1.0.0 that is reserved for a major bump.

- **0.1.0 → 0.1.1** — a fix. Safe to take blindly.
- **0.1.1 → 0.2.0** — new things, possibly a setting that needs attention.
- **1.0.0** — the point at which configuration, the HTTP API and the data on disk are promised
  stable enough that breaking them needs a major version and a migration note.

The per-component dates in `internal/changelog/changelog.json` (`2026.11.132` and friends) stay as
the internal build history. They are not what users install and are never used as a tag.

## What a tag produces

Pushing `v0.1.0` (and only if it matches `VERSION`) publishes:

| Where | What |
| --- | --- |
| Container | `:0.1.0`, `:0.1`, `:latest` |
| Releases | `mochil-0.1.0-<os>-<arch>` archives plus `SHA256SUMS` |
| Binary | `mochil -version` prints `0.1.0 (abc1234)` |

`edge` is separate: every push to `main` republishes it, and it is never tagged `latest`. That is
the channel for "I want to try what is in the tree", and it exists so `latest` can stay honest.

## Steps

1. Update `VERSION`.
2. Add the user-facing notes to `CHANGELOG.md` under the new number — written for someone who
   installed the previous release, not for someone reading the diff.
3. Commit, then tag and push:

   ```bash
   git tag -a v0.1.0 -m "Mochil 0.1.0"
   git push origin main v0.1.0
   ```

4. The workflow verifies the tree, builds five binaries, builds the multi-arch image, proves the
   **published** image installs from empty volumes, then opens the GitHub release.

## What users are told to do

```yaml
image: ghcr.io/kevindherdt/mochil:0.1      # safe updates within 0.1
# image: ghcr.io/kevindherdt/mochil:0.1.4  # pinned exactly
# image: ghcr.io/kevindherdt/mochil:latest # whatever is newest
```

Updating is always the same, and never loses data because the three folders are mounted:

```bash
docker compose pull && docker compose up -d
```

## Why the tag must match VERSION

The workflow fails before building anything if they disagree. A release labelled `0.2.0` while its
`VERSION` says `0.1.9` is worse than no release: users pin a number, and the number has to mean the
same thing in the image, the archive and `mochil -version`.
