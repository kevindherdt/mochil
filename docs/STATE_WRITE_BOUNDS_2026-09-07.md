# Persistent state write/read limits

The latest source hardening aligns these writers with their existing decoder
budgets. Previously they could write a state file that their next startup would
reject solely for size.

| State | Existing read limit, now also applied on write | Encoding |
| --- | --- | --- |
| Audio queue and playlists | 4 MiB per store file | Compact JSON |
| Audio progress | 64 MiB | Compact JSON |
| Video progress | 64 MiB | Compact JSON |
| Reading progress and history | 64 MiB | Existing two-space-indented JSON |

Each remains an atomic temporary-file replacement. An over-budget encoded
document returns an error before replacing the previous state; existing store
rollback paths retain their prior in-memory state. No old user data was trimmed,
deleted, migrated, or manually rewritten. These limits apply across the whole
store, not separately to each account.

Audio stores share their filesystem save owner; video and reading keep their
existing save lifecycle and invoke the bounded encoder directly. The shared
encoder tests now cover exact-byte acceptance and one-byte rejection with Unicode,
JSON escaping, indentation, and the terminating newline. This preserves existing
serialization rather than changing it to fit a smaller budget. Existing video
and reading failed-persistence tests cover rollback; audio tests additionally
exercise oversized-save rejection and temporary-file cleanup.

This is a file-integrity bound, not a guarantee of bounded encoder allocation:
Go's JSON encoder can build the encoded document in memory. Long-term growth
beyond these existing budgets needs a separately designed storage migration,
not silent truncation. Other store families are outside this bounded pass.

Deployed September 7 at 16:13 UTC in running image
`sha256:ec0b52d3a2f59bcda060a106f4db5ce697286b20d8d30cb7dc5d97a3c91fe8b7`.
The container reports healthy, zero restarts, and `/healthz` returns `ok`.
Only opdshelf was recreated. The mounted Android APK remains unchanged at
SHA-256 `559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6`.
