# Bounded vault reads during mutations

Status: post-0.16.75 worktree, not deployed.

The writer previously checked the file's size with Lstat and then used an unbounded ReadFile. An external writer could grow the file after that check. Update, move, and trash revision checks now use a read limited to the existing 8 MiB page budget plus one detection byte. Oversized data is rejected rather than returned as a truncated page; exact-limit and empty pages remain valid.

Tests cover exact boundaries, propagated I/O failures, and a synthetic never-ending reader, which consumes exactly 8 MiB + 1 byte before rejection. This bounds input consumption, not total Go allocation to exactly that amount. Vaultwriter tests, its race run, and the web package tests passed.

This is not a complete external-filesystem race solution: path replacement between checks and operations, and simultaneous external edits after final revision checks, still require a separate design review. The writer mutex coordinates operations through this writer, not arbitrary external editors. No user vault files or running containers were changed.

## Move and trash durability ordering

Both hard-link-based moves and trash operations now share `completeLinkedMove`: sync the destination directory before unlinking the original, then sync the original directory. Previously the original could be unlinked and its directory synced before the destination directory was synced. The new ordering reduces the crash window in which the original removal is durable but the destination entry is not.

A destination sync failure leaves both names and reports failure. An unlink failure also retains the destination; no cleanup deletes the recovery copy. This deliberately favors recoverable duplicates over losing a page. It is not an atomic multi-directory transaction and does not resolve arbitrary concurrent external edits.

Tests inject destination sync failure, verify original preservation, check same-directory sequencing, and verify cross-directory sync ordering. Existing move/no-clobber and trash tests still pass. Race-enabled vaultwriter and web package tests passed. Changes are not deployed.
