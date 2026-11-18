# Publication normalization helpers

Status: normalization optimization and identifier matching follow-up published
in the server-only grouping-performance batch at 17:19 UTC, 2026-09-08.
Earlier pending statements below describe development chronology.

Publication role classification and identifier normalization now retain their
fixed strings.Replacer instances rather than constructing them per publication.
Role precedence, phrase matching and identifier output are unchanged.

Two 100ms microbenchmark runs of role separator replacement measured:

- Previous: 383–410 ns/op, 576 bytes/op, five allocations.
- Shared: 103–104 ns/op, 96 bytes/op, two allocations.

This measures only separator replacement, not a full scan or metadata review.
Full Go tests, vet, focused parallel race checks and module gate passed.
Logs: `/tmp/bobobooks-publication-normalizers-{bench,go,vet}.log`.

## Separate matching issue found

The existing identifier normalizer compacts any value whose resulting length
is 10 or 13, without verifying ISBN syntax. For example `provider:ab-cd` becomes
`provider:abcd`. This performance-only change preserves that output explicitly
in a regression. Follow up with provider-aware normalization and edition-group
tests before changing matching semantics; identifiers must not be conflated
simply because punctuation removal happens to produce an ISBN-like length.

## Matching follow-up implemented

Identifier normalization now removes separators only for ISBN-shaped values:
ten digits with an optional terminal X, or thirteen digits. A single recognized
ISBN prefix is removed only when its remaining value has that shape. Invalid
or other-provider values retain their normalized prefix and punctuation.
This does not validate checksums or equate ISBN-10 with ISBN-13.

Preferred-format backend tests prove that `provider:ab-cd` and `provider:abcd`
remain separate publications, as do invalid prefixed identifiers and unprefixed
lookalikes. Valid-shaped ISBN prefix/separator variants still collapse according
to the configured format preference. No files or embedded metadata are rewritten.

Full Go tests, vet and module gate passed. Logs:
`/tmp/bobobooks-provider-id-{go,vet}.log`. This fix remains source-only after
0.16.85, alongside the allocation improvements.
