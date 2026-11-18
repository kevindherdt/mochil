# Android wiki search-order cache

Status: source-only improvement after 0.16.79. No new APK or deployment.

`WikiTreeIndex.search` previously sorted matching pages on every query. Its comparator computed lowercase paths repeatedly. The index now lazily creates a stable path-ordered page list on first nonblank search, with one lowercase key per page during construction. Subsequent queries filter that list without sorting. A replaced vault tree owns a new cache; no global cache or cross-library reuse is introduced.

The existing Compose caller performs both tree construction and search on `Dispatchers.Default`. Folder expansion and blank queries still use natural hierarchical order and do not initialize the search cache. Stable case-folded ties, Unicode matching, collapsed-page discovery and attachment exclusion remain unchanged.

Tradeoff: the first nonblank search sorts all pages rather than just the matching subset, and the tree retains one extra page-reference list. Later queries avoid O(k log k) sorting and comparator lowercase allocations, retaining O(n) filtering. No device timing or percentage improvement is claimed.

Verification: Direct and Store `WikiTreeIndexTest` suites executed successfully (10s Gradle run). Repeated-query tests compare with the prior algorithm across Unicode, case-folded ties, missing results and blank-query restoration; a fresh index cannot reuse old search results. Module architecture gate passed. Existing UI is unchanged.

## Host-JVM benchmark

`WikiSearchBenchmarkTest` is opt-in via `MOCHIL_BENCHMARK_WIKI=1`; ordinary tests do not impose timing thresholds. It compares the prior sorting policy with the cached index over 5,000 synthetic notes, checks matching results, warms each path ten times and reports the median of nine batches of twenty searches. The measured unit is one search, excluding tree construction and transport/rendering. Gradle run completed successfully in 3s on this development host.

| Query | Matches | Previous median | Cached median | Ratio |
| --- | ---: | ---: | ---: | ---: |
| Note | 5,000 | 1.138 ms | 0.152 ms | 7.50× |
| Note 49 | 111 | 0.325 ms | 0.244 ms | 1.33× |
| missing | 0 | 0.269 ms | 0.216 ms | 1.24× |

The initial cold comparison executing **both** implementations plus equality checking took 37.318 ms; it does not isolate the new lazy-cache startup cost. This single host-JVM run is not an Android-device latency, allocation, frame-rate or full-page speedup claim. Small differences, especially the no-result case where neither implementation sorts matches, may reflect JVM effects and measurement noise. The extra cached reference list remains the memory tradeoff.

Reproduce from `android/` with `MOCHIL_BENCHMARK_WIKI=1 ./gradlew :app:testDirectDebugUnitTest --tests '*WikiSearchBenchmarkTest' --rerun-tasks --console=plain`. The rerun flag prevents an up-to-date test result from being mistaken for a fresh measurement. Output is in the JUnit XML `system-out` for this test class.
