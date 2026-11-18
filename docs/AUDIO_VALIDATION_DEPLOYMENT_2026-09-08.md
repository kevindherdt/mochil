# Audio validation server deployment

Published 2026-09-08T18:22:19.161749336Z as
`opdshelf:audio-validation-20260908`, image
`sha256:26e0ccf6b4ce4140f282d0d5e73d0ae48cf56e70b04ce5b361cb53545a2ce230`.

Includes finite/nonnegative podcast duration parsing in its domain module and
deterministic normalized audio title/series ordering in a focused sort module.
All preceding 0.16.86 changes remain included.

Full Go tests, vet, 564 frontend tests and module checks passed. An isolated
authenticated candidate passed update-info/download/hash/HEAD/range and
conditional playlist mutation checks. Fixture container was stopped.
Logs: `/tmp/bobobooks-audio-validation-{go,vet,web,update,playlist,image}.log`.

Live health endpoint and Docker health pass with zero restarts. All ten
libraries loaded; the known malformed EPUB remains the sole scan warning.
Android remains 0.16.86/code 247, live APK SHA-256 unchanged:
`67a31e56857fe309fa1d065917bbdab3867aeeb6836f1543d2b2dfe835bc394d`.
Rollback image `opdshelf:android-0.16.86` remains available. No personal media
was modified and no unrelated service was restarted.
