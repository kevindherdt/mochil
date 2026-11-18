# Reading checkpoint performance

Measured on September 7, 2026 in the development container, Linux amd64,
AMD Ryzen 5 PRO 4650G. Synthetic in-memory accounts contain five history
snapshots per item, a text anchor and a Readium locator.

The operation acknowledges an equivalent stale checkpoint. The comparison
adds back the whole-store rollback snapshot previously taken before revision
checks. It is a reconstructed cost comparison, not a checkout of an older
release. Neither path writes files or includes HTTP, network or UI time.

| Entries | Current time/call | With former snapshot cost | Current allocated bytes/call | With former snapshot cost |
| --- | --- | --- | --- | --- |
| 100 | 476–480 ns | 279–284 µs | 448 B | ~405 KB |
| 5,000 | 530–556 ns | 27–30 ms | 448 B | ~20.4 MB |

Allocation counts: current 4/call; comparison 2,016/call at 100 entries,
100,044/call at 5,000 entries. These are cumulative allocations per operation,
not resident memory or a promise of equivalent whole-app savings.

## Reproduce

```sh
go test ./internal/reading -run '^$' -bench '^BenchmarkEquivalentCheckpointLargeHistory$' -benchmem -benchtime=200ms -count=2
```

The benchmark lives in internal/reading/sync_benchmark_test.go. Accepted
mutations still use the existing rollback and persistence path. Conflict
semantics and returned-position isolation remain covered by regression tests.
The full Go test suite passed after the sync changes; reading/audio/video/web
race suites passed during their respective implementation checks.

Deployed September 7, 2026 at 14:42 UTC after the full Go suite and isolated
Compose startup/recovery tests passed. Live health check passed with zero
restarts and all ten libraries ready.
