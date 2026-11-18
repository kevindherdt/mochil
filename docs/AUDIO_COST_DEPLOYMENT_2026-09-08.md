# Audio cost reduction deployment

Published 2026-09-08T18:31:27.137282065Z, image
`sha256:9d54a47167bc3e166c5175e1e0062ccd0e9b6615fa2fba2058427394ee576fe5`,
tag `opdshelf:audio-cost-20260908`.

Includes prepared keys for title-sorted audio pages/navigation and omission of
unused whole-account progress snapshots from unfiltered audio pages. Behaviour
and scoped benchmark evidence are in `AUDIO_TITLE_SORT_COST_2026-09-08.md` and
`AUDIO_HISTORY_SNAPSHOT_COST_2026-09-08.md`; no whole-app speedup is claimed.

Full Go tests, vet, module and source readiness gates passed. The candidate
passed authenticated APK metadata/full-hash/HEAD/range checks and conditional
playlist mutation checks. The temporary fixture was stopped.
Logs: `/tmp/bobobooks-audio-cost-{go,vet,image,update,playlist,source-gate}.log`.

Post-deployment health endpoint passes, zero restarts, all ten libraries loaded.
The only scan warning remains the previously known malformed EPUB. Live APK
remains 0.16.86/code 247 with matching hash:
`67a31e56857fe309fa1d065917bbdab3867aeeb6836f1543d2b2dfe835bc394d`.
Prior `opdshelf:audio-validation-20260908` and all release APK archives remain
available for rollback. No personal media or unrelated containers were changed.
