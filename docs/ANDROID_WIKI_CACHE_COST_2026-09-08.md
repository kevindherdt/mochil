# Native wiki offline image-cache cost

Status: published in Android 0.16.83 on September 8, 2026. No UI changes.

The cache lookup previously enumerated and hashed all image entries even when no
cached files existed. It now returns immediately in that case, avoiding one SHA-256
calculation per image entry (and the traversal) for an empty cache.

Existing cache filenames remain SHA-256 over the same UTF-8 identity fields with
the same separators, followed by `.bin`. Lowercase hex encoding now appends two
characters per digest byte instead of creating 32 string formatters per filename.
This keeps existing downloads compatible; no cache migration or deletion is added.

The compatibility regression compares output against the previous algorithm with
empty strings, non-ASCII paths, emoji, whitespace and embedded separators. Full
Direct and Store unit suites passed. The module-size gate passed. This is a
reduction in known work, not a measured device latency or RAM percentage claim.

The broader offline audit still needs to cover attachment support beyond embedded
images and refresh/retention behavior; this change does not claim full vault parity.

## Partial-refresh image retention

The mirror previously planned image downloads and pruning from only the newly
fetched pages, although the encrypted page store merges those with older saved
pages. A partial refresh could consequently prune images used by retained notes.

Planning now uses the snapshot read back after the save, including retained pages
and the persisted entry index. If that snapshot cannot be read back, the operation
fails before image pruning. The existing image count/byte limits still apply; this
does not promise every retained image fits within the bounded cache.

Counting newly requested pages present in the snapshot now uses a hash set rather
than a nested page scan. The regression test passes a partial refresh through the
real merge, bounded encoding, decoding and asset-planning functions and checks that
both the retained and new note's image are included. It does not emulate Android
encrypted disk persistence or device-level storage failures.

Both full Android flavor unit suites and the module-size gate passed after this
follow-up (`/tmp/bobobooks-wiki-retained-assets-tests.log`). Pending batch release.
