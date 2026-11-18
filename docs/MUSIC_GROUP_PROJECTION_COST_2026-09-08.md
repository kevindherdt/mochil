# Full music grouping allocation audit

Status: server-only batch published; Android remains 0.16.85. Earlier pending
statements below describe development chronology.

## Deployment verification

Published at 2026-09-08T17:19:31.861101359Z with image
`sha256:3881396e71f79ac13b68f70900146e662ed5e53792de6dd7b5fa0bf5d1513e7b`,
retained as `opdshelf:grouping-performance-20260908`. Container healthy, zero
restarts, all ten libraries loaded; only the known malformed EPUB warning.
The mounted APK still matches the verified 0.16.85 Direct hash.

Before deployment, 563 frontend tests, playlist browser checks at 1440/390px,
and candidate authenticated update (full hash, HEAD, range) and playlist
revision checks passed. Disposable fixture stopped. Previous image retained
as `opdshelf:android-0.16.85`; no media or unrelated containers changed.
Build log: `/tmp/bobobooks-grouping-performance-image.log`.

Added a reusable 10,000-track benchmark covering identity construction, merging,
filtering and sorting for albums/artists. It excludes store snapshot copying,
JSON serialization and network latency.

The CPU profile exposed heavy allocation/GC in album grouping. Each track's
disc-folder check constructed a strings.Replacer with identical rules. That
helper is now shared; the case, whitespace, separators and positive-disc-number
rules are unchanged. No cross-request catalogue cache is introduced.

| Album projection | Before bytes/op | After bytes/op |
| --- | --- | --- |
| All albums | 69,691,657 | 2,325,934 |
| Contributor search | 69,925,325 | 2,564,928 |
| No matches | 69,918,460 | 2,557,276 |

This is roughly 96% fewer allocated bytes for the synthetic album projection.
Observed times moved from 31–37ms to 9.5–9.9ms, but the baseline also collected a
CPU profile; treat timing as indicative rather than a controlled speedup claim.
Artist allocation was essentially unchanged, as expected for its separate path.
These are allocation totals, not retained memory or whole-server RAM savings.

Logs: `/tmp/bobobooks-group-projection-bench.log`,
`/tmp/bobobooks-group-projection-after.log`.
Profile: `/tmp/bobobooks-group-projection.cpu`.
Full Go tests and vet passed: `/tmp/bobobooks-disc-folder-{go,vet}.log`.
Parallel disc-folder cases additionally exercise the shared helper under the
race detector. Existing edition-folder identity tests preserve grouping behavior.

## Shared audio ownership

The same per-call construction also existed in music metadata inspection.
Both implementations have now been replaced by `audio.IsDiscFolder` in the
focused `internal/audio/disc_folder.go` module. This owns only disc-directory
recognition; callers keep their own path handling and grouping responsibilities.
The shared normalizer's parallel test moved to the owning audio package.

Full Go tests, vet, the focused audio race test and module gate passed after
extraction. Existing metadata-inspection and web edition-identity tests cover
both callers. Logs: `/tmp/bobobooks-shared-disc-{go,vet}.log`.
No numeric performance claim is made for metadata inspection without a separate
report-level benchmark. The changes remain unpublished after 0.16.85.
