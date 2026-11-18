# Wiki ordering responsibility and deep hierarchy

Status: source changes after 0.16.84, pending a later batch.

Natural comparison and hierarchy ordering now live in WikiOrdering.kt rather than
the data-model file. This separates a cohesive algorithmic responsibility; it is
not a split to meet an arbitrary line target.

Depth-first traversal uses an explicit iterator stack. Deep folder paths therefore
do not consume one Java/Kotlin call frame per folder. Single-entry sibling lists
skip tokenization/sorting entirely, and parsed sibling keys are discarded before
descending. Existing folder-first natural ordering and stable path ties remain.
Blank root entries are ignored instead of recursively reopening the root.

New tests cover 2,000 nested folders plus a leaf, blank root input and resuming
sibling branches in correct order. Existing mixed-name and folder tests also pass.
Both full Android debug unit suites passed in 15 seconds; module gate passed.
Log: `/tmp/bobobooks-wiki-deep-tests.log`.

This bounds call-stack usage in the ordering algorithm, not total vault memory or
all other hierarchy processing. No UI layout, personal data or deployed APK changed.
