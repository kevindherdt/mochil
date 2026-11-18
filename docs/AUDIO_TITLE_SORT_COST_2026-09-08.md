# Prepared audio title sorting

Status: published at 18:31 UTC; see `AUDIO_COST_DEPLOYMENT_2026-09-08.md`.

Title-ordered audio requests repeatedly normalized both titles inside sort
comparisons. `sortAudioItems` now prepares title keys once per request for title
sorts and default non-podcast title ordering. A parallel key slice follows item
swaps, preserving natural numeric comparison and stable ID ties. Date, artist,
album and series modes still use the existing comparator; they do not pay for
unused title keys. No cross-request cache or invalidation policy was added.

The main audio endpoint, track alphabet navigation and audiobook endpoint use
the shared entry point. The focused helper is separate from HTTP plumbing and
the multi-mode ordering policy.

## Benchmark

5,000 synthetic tracks, same source copy and stable sorting in both paths,
three runs on AMD Ryzen 5 PRO 4650G. Go benchmark output:
`/tmp/bobobooks-audio-title-benchmark.log`.

| Path | Time per sort | Allocated bytes | Allocations |
| --- | --- | --- | --- |
| Repeated normalization | 26.75–26.94 ms | 4,618,152–4,618,155 | 160,608 |
| Prepared title keys | 9.69–10.82 ms | 2,209,968–2,209,975 | 5,003 |

About 60–64% less sort time, 52% fewer allocated bytes and 97% fewer allocations
in this benchmark. These numbers include copying the input but not catalogue
filtering, HTTP serialization, network transfer or browser rendering; they are
not a promised whole-page speedup.

## Verification

240 shuffled mixed-input cases compare complete item order against the existing
comparator: eight sort modes, three library kinds and ten seeds. Includes
article/case/numeric ties, empty names, Unicode, dates, discs and track numbers.
Full Go tests, vet and module gate passed. Logs:
`/tmp/bobobooks-audio-title-go.log`, `/tmp/bobobooks-audio-title-vet.log`.
