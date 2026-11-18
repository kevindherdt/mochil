# Music group sorting cost

Status: published in the server-only grouping-performance batch at 17:19 UTC,
2026-09-08. Android remains 0.16.85. See the deployment record in
`MUSIC_GROUP_PROJECTION_COST_2026-09-08.md`.

Album/artist sorting now prepares the existing catalogue SortKey once per group
and swaps keys alongside groups. It retains the same natural comparison and
stable ID tie-break, with no change to group identity or client layout.

Thirty shuffled regression cases compare exact output with the prior algorithm,
including articles, case, whitespace, equal numeric values and non-ASCII names.

Synthetic 5,000-group benchmark, two 100ms runs on the development host:

| Algorithm | Time per sort | Allocated bytes | Allocations |
| --- | --- | --- | --- |
| Previous | 19.48–20.65 ms | 3,377,160–3,377,176 | 183,418 |
| Prepared keys | 4.38–4.52 ms | 604,336–604,343 | 5,003 |

Both include the same input-slice copy. This is approximately 77–79% lower sort
time and 82% fewer allocated bytes in this synthetic case, not a claim about
whole-page latency or steady server RAM. No cache or stale-result lifetime is
introduced. Command: `go test ./internal/web -run
TestAudioGroupSortPreparedKeysPreserveOrder -bench '^BenchmarkAudioGroupSort$'
-benchtime=100ms -count=2`.

Full Go tests, vet and module gate are recorded in
`/tmp/bobobooks-group-sort-{go,vet}.log` after the focused check.
