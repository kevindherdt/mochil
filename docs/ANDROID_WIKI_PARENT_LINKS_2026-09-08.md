# Compact wiki hierarchy visibility

Status: source change after 0.16.84; pending a later batch.

WikiTreeIndex previously retained lists of all ancestor path strings, shared only
between siblings. Deep vaults still accumulated many path copies and checked every
ancestor on each expansion projection.

The naturally ordered tree now retains one integer parent index per row. Visibility
is computed in one forward pass using already-computed parent visibility, with at
most one expansion-set lookup per visible folder. Parent maps used at construction
are not retained. This removes ancestor-list/path duplication from the index.
The existing lazy path-sorted search cache is unchanged.

Verification: 100 randomized expansion sets against an independent all-ancestor
reference filter; 2,000-level hierarchy expanded fully, collapsed halfway, and
searched while collapsed. Both complete Android debug unit suites passed in 13s;
module gate passed. Log: `/tmp/bobobooks-wiki-parent-tests.log`.

Space for retained parent links is O(number of rows), and each visibility pass is
O(number of rows), excluding string hash costs. Total vault paths/search data still
consume memory. No whole-app RAM/frame-time claim or device measurement is made.
