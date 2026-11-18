# Wiki inline parsing

The previous inline parser performed six independent `Regex.find` searches over the remaining text after each span, then allocated candidate triples and compared positions to select a winner. Patterns absent from a long formatted paragraph repeatedly searched the same tail.

`WikiInline.kt` now owns inline tokenization with one ordered alternation. Alternative order preserves tie precedence: wiki links, HTTP(S) links, bold, strikethrough, code, italic. Cursor gaps, label fallback, whitespace trimming, and unsupported-syntax behavior remain unchanged. Block parsing stays in `WikiMarkdown.kt`; this is responsibility-based separation, not a line-count target.

Verification uses a frozen copy of the prior parser as an oracle for 17 × 17 × 5 = 1,445 fragment combinations, including nested-looking markup, malformed fragments, Unicode, and code. A separate 5,000-bold-span test checks output order and segment count. Both full Android suites passed (705 tests per variant), and module checks passed.

This removes repeated independent tail searches/candidate-list allocations but does not establish a universal regex complexity bound or measured device latency/RAM percentage. No new formatting features or visual changes are introduced. Changes remain unpublished after 0.16.77.

## Wide table rows

Table row parsing previously built every cell before truncating the result to 20 columns. It now returns as soon as the twentieth complete cell is found, without tokenizing/allocating discarded columns. Existing line trimming can still copy the source string; this is a bound on cell accumulation, not constant memory for an arbitrarily large input line. The existing 20-column rendering policy is unchanged.

Tests cover 100,000 surplus columns, escaped pipes, and the exact column limit with a trailing backslash. Both Android suites passed (708 tests each), and module checks passed. This remains part of the next unpublished batch.
