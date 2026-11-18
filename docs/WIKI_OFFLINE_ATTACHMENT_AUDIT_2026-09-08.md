# Android wiki offline attachment audit

Scope: image mirror orchestration, cache ownership, size budgets and read-back
behaviour. This is not a complete wiki security or physical-device review.

## Finding corrected in source

`WikiOfflineAssetStore.cachedUrls` previously exposed matching files based on
existence alone. Download reuse and writes checked byte budgets, but a later
empty, oversized or over-budget cache could bypass those checks on reopening.

The read path now uses the same bounded-name policy: at most 32 images, 3 MiB
per file and 24 MiB in total, scoped to the requested owner/vault and exact
entry identity/revision/path. Invalid files are omitted without deleting user
content or forcing a network request. URL conversion remains separate from
file selection, allowing filesystem regression tests without Android URI stubs.

Two new tests cover empty/oversized/stale entries, owner/vault isolation,
non-image entries and both aggregate/count limits before any pruning pass.
Both complete Android debug unit suites and the module-size gate passed.
Log: `/tmp/bobobooks-wiki-read-bounds-tests.log`.

## Existing safeguards inspected

- App-private no-backup storage, hashed owner/vault and entry identities.
- Atomic replacement where supported, serialized merge/prune across instances.
- Four-request maximum mirror concurrency and bounded retained image batches.
- Network byte limits enforced both against announced size and actual reads;
  response MIME must be an image. This is not image-content signature validation
  or protection against every large decoded-pixel image.
- Cancellation propagates rather than being reported as an optional missing image.
- Broader offline non-image attachment mirroring is still absent; no such
  capability is claimed by this patch.

Delivery: published in 0.16.86/code 247, alongside the TV changes.
See `ANDROID_0_16_86_RELEASE_VERIFICATION.md` for verified release evidence.
