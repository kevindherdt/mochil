# Audio queue persistence hardening

Source review found that `QueueStore.Put` and persisted-state decoding enforced
the 10,000-entry limit, but `Add` validated only its single input entry. Incremental
adds could therefore save an oversized queue rejected on the next server start.
All queue mutations now validate their complete proposed result before saving.
A duplicate add at capacity still succeeds without changing the queue.

The shared mutation path also recognizes an unchanged ordered queue before
copying account state, encoding JSON, creating/syncing a temporary file, and
renaming it. Repeated Put and removal of an absent entry therefore avoid disk
writes. Nil and empty queues are treated as equivalent; an empty no-op does not
create an account. Real ordering changes still persist, and persistence failures
leave memory untouched. This is not cross-device conflict resolution: competing
different queue replacements retain the existing last-write behavior.

Tests fill a persisted queue to capacity, reject the next incremental add, and
reopen the saved file successfully. Other tests use an unwritable replacement
target to prove no-op writes are skipped while actual reorders attempt storage;
returned queue isolation is also covered. The full audio package passes with
the race detector. No throughput, battery, or end-to-end latency gain is claimed.

User media and existing runtime queue files were not manually edited.

## Playlist follow-up

Playlist incremental adds already enforce the limit. Identical normalized-name
and ordered-entry saves, and renames to the exact existing name, now skip disk
writes too. Case-only name changes still persist: case-insensitive lookup is not
used to decide whether the displayed name changed. Tests cover those distinctions,
failed reorder isolation, missing-playlist rename errors, and preservation of
an intentionally empty playlist. The complete Go suite, audio race tests, and
module-size gate pass after the combined changes.

## Deployment

Deployed September 7 at 15:55 UTC with the web queue focus fixes. Running image
`sha256:6054aadd6db2b264ff952a6fbc39b680a79f9ada0db68e9e3603c5d54c110fab`
reports healthy, zero restarts, and `/healthz` returns `ok`. Only opdshelf was
recreated. Mounted Android 0.16.72 APK SHA-256 remains
`559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6`.

## Controlled queue-save measurements

September 7 benchmark on Go 1.25.0, Linux amd64 / Ryzen 5 PRO 4650G, three 200 ms runs
per case. Temporary files were on tmpfs: this measures a memory-backed filesystem,
not the production volume's physical-disk latency. Incoming IDs use independently cloned strings to avoid benefiting
from pointer-equal string comparison. Setup and request decoding are excluded.

| Queue entries | Unchanged Put, median | Forced persistence, median |
| --- | --- | --- |
| 100 | 0.676 microseconds | 44.286 microseconds |
| 1,000 | 6.993 microseconds | 139.783 microseconds |
| 10,000 | 71.137 microseconds | 1,148.821 microseconds |

Unchanged Put allocated zero bytes/objects per operation in all runs. Forced
persistence allocated 20 objects; byte counts varied with encoder-buffer pool
reuse. The forced comparison uses today's bounded writer while bypassing the
unchanged check, not an old revision or a replayed HTTP request. It includes
temporary creation, encoding, file sync, and rename but not the previous
mutation path's account-copy work. Results depend on the local filesystem and
cache; they do not measure browser latency, Android battery, concurrent accounts,
network overhead, or cold-disk behavior. Tests use temporary files only.

Reproduce with `go test ./internal/audio -run '^$' -bench BenchmarkQueueUnchangedSave -benchmem -benchtime=200ms -count=3`.
