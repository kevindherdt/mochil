# Wiki natural ordering cost

Status: source change after 0.16.84, pending a later batched APK.

The hierarchy already cached folder expansion. Its initial sibling sorting still
parsed both labels with a regular expression on every comparator invocation.
Sorting now prepares name segments once per sibling group, retaining existing
folder-first order, path tie-breaking, case handling, leading zeros and arbitrarily
large integer comparison. Prepared segments are temporary, not a persistent cache.

## Verification and measurement

Both full Direct/Store debug unit suites passed in 16 seconds, including existing
hierarchy tests and shuffled mixed-name equivalence checks. Module gate passed.

Opt-in host-JVM fixture: 5,000 shuffled sibling pages, five warmups, eleven samples
with alternating old/new order; medians include equality checks. The baseline
uses the previous per-comparison tokenization; new path includes hierarchy setup.

| Variant | Previous sort | Prepared names | Reduction |
| --- | ---: | ---: | ---: |
| Direct test JVM | 49.90 ms | 7.09 ms | 85.8% |
| Store test JVM | 44.34 ms | 7.12 ms | 83.9% |

This is a synthetic JVM sorting improvement, not measured Android frame time,
whole-vault load speed, or memory reduction. No timing threshold gates tests.
Command: `MOCHIL_BENCHMARK_WIKI_ORDER=1 ./gradlew :app:testDirectDebugUnitTest :app:testStoreDebugUnitTest`.
Log: `/tmp/bobobooks-wiki-order-tests.log`; exact values in test XML system-out.
