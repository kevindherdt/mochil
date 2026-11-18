# Bounded music artwork sibling lookup

Status: deployed September 8, 2026 at 16:24 UTC as a server-only update.
Image `sha256:ef36f1bc8835edc5a7e3b3ee799402a36ad8b8d1b4236936fec65956f1069f2c`,
started `2026-09-08T16:24:07.997914326Z`, healthy, zero restarts, ten libraries ready.
The Android download remains the verified 0.16.84 archive, unchanged SHA-256
`60c3e0bbe6f076792e6af42d54cdba160e50291c83215671ef8a5cdf680c3c9a`.
Previous image `opdshelf:android-0.16.84` retained for rollback.

Full Go tests/vet, audio race tests, 563 frontend tests and module gate passed.
The added concurrent publication regression exercises 400 lookups while scans
replace the catalogue. Candidate authenticated update/hash/HEAD/range and playlist
revision checks passed; disposable fixture stopped. Personal media unchanged;
the existing malformed EPUB is still the sole scan warning.

ArtworkSiblings previously cloned the complete audio catalogue before selecting
a few sibling candidates. It also accumulated every known-cover sibling before
truncating to the requested limit. The new implementation reads one consistent
snapshot under the store read lock, bounds candidate accumulation, exits when
enough known covers are found, and deep-copies only returned items. No filesystem
or decoder work occurs under that lock. Worst-case catalogue traversal remains O(n).

Tests preserve known-cover priority, stable candidate order, requested limits,
missing-item behavior and detached chapter slices. Audio race tests and audio/web
tests passed; module gate passed.

## Synthetic benchmark

10,000 tracks, 20 tracks per album, requested limit four; Go host amd64,
GOMAXPROCS=1, three 100ms samples. Medians below compare the previous algorithm
with the current implementation. These short, non-interleaved samples are
directional, not whole-app latency or steady-state RAM measurements.

| Album position | Previous | Bounded | Previous bytes/op | Bounded bytes/op |
| --- | ---: | ---: | ---: | ---: |
| Near beginning | 4.299 ms | 0.00394 ms | 4,094,848 | 5,376 |
| Near end | 4.924 ms | 0.487 ms | 4,094,848 | 5,376 |

Allocation reduction is about 99.87% in this fixture. End-of-catalogue lookup
still scans preceding tracks but avoids cloning their metadata. Concurrent scan
publication may wait for this bounded read-side traversal; no device/browser
cover-load timing has been measured.

Logs: `/tmp/bobobooks-artwork-siblings-{race,tests,bench}.log`.
