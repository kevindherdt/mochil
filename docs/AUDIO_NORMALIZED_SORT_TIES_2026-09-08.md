# Audio normalized sort ties

Status: published at 18:22 UTC; see `AUDIO_VALIDATION_DEPLOYMENT_2026-09-08.md`.

Regression tests first failed against the old comparator: Song/song/SONG kept
input order across all seven tested sort modes instead of reaching the ID
tie-break. Equivalent series names such as The Saga 02/Saga 2 also returned
early and skipped volume ordering.

The comparator now distinguishes less, greater and equivalent natural labels.
Equivalent titles reach stable item-ID ordering; equivalent series names reach
volume/disc/track ordering. Existing date, artist and album priorities remain.
The policy lives in `audio_order.go`, separately from HTTP parsing, progress
projection and response encoding. No new sorting framework was introduced.

Tests cover 20 shuffled inputs per sort mode plus article/numeric-equivalent
series names. Existing series, podcast ordering, cursor and navigation tests
continue to pass. Full Go tests, vet and module gate passed:
`/tmp/bobobooks-audio-order-tests.log`, `/tmp/bobobooks-audio-order-vet.log`.

This provides deterministic tie ordering, not a measured performance gain or
an assertion that every possible catalogue mutation preserves an old cursor.
No media files or installed APKs changed.
