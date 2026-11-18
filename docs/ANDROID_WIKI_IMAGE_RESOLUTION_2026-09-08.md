# Native wiki image resolution

Pending after 0.16.78; no APK or deployment.

`WikiAssetIndex` now follows the corrected web image selection rules: relative path before root, explicit dot-relative references do not fall back to root, traversal above the vault is rejected, and ambiguous basename/case-folded matches remain unresolved rather than selecting whichever file scanned first. Unsafe schemes and controls are rejected. The scheme expression is compiled once, and indexed lookup no longer sorts candidate matches.

The shared index serves native rendering and offline reference selection, so these no longer resolve the same note differently merely due to catalogue ordering. Existing cached files are not deleted or rewritten by this change.

Tests cover local/root order reversal, explicit relative paths, ambiguous paths and unsafe targets. A pre-existing test required the old root-first-by-index behavior; it was revised to require relative priority. Both full Android debug suites passed after that correction (11 seconds), and module checks passed. Physical image rendering/offline acceptance remains unverified.
