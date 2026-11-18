# Music server benchmark checkpoint

Date: September 8, 2026. Go 1.25.0, Linux amd64, Ryzen 5 PRO 4650G.
Synthetic fixtures only; no personal queue or media files modified.

## New measured change

`QueueStore.Put` validated every entry before locking, then `putLocked` validated
the same entries again. Removed the redundant outer pass. The shared locked
mutation boundary still validates every queue before comparing or changing state.
Invalid requests now acquire the mutex before rejection; validation remains
bounded by the existing 10,000-entry limit.

Median of three 100ms benchmark samples with `GOMAXPROCS=1`:

| Unchanged queue | Before | After | Time reduction |
| --- | ---: | ---: | ---: |
| 100 tracks | 0.734 µs | 0.642 µs | 12.5% |
| 1,000 tracks | 6.904 µs | 6.486 µs | 6.1% |
| 10,000 tracks | 73.866 µs | 62.192 µs | 15.8% |

Every unchanged case retained 0 B/op and 0 allocations/op. Short, non-interleaved
samples on a shared host are directional evidence, not statistical significance
or end-to-end app speed claims. No Android frame-time/RAM improvement is inferred.

Command: `GOMAXPROCS=1 go test ./internal/audio -run '^$' -bench BenchmarkQueueUnchangedSave -benchmem -benchtime=100ms -count=3`.
Before: `/tmp/bobobooks-music-benchmarks-20260908.log`.
After: `/tmp/bobobooks-queue-validation-after.log`.

## Existing optimizations rechecked

At 1,000 playlists of 100 entries, the current single-playlist Get median was
10.458 µs / 3,456 B / 1 allocation, versus 2,719.894 µs / 3,529,432 B / 3,026
allocations for cloning/sorting the entire list then finding one. This compares
two current code paths, not a new change or a historical release checkout.

At 10,000 queue entries, forced persistence after the change had a median
1.229 ms versus 62.192 µs for the unchanged fast path. The control deliberately
bypasses the fast path using today's writer; it is not a former implementation.

## Verification and delivery

Audio/web tests and audio race tests passed. A new regression proves that empty
account, missing item/library identity and oversized queues remain rejected without
changing saved state. The final race run includes that test. Deployed in the
web/server-only update at 15:24 UTC on September 8. Android remains 0.16.83.
