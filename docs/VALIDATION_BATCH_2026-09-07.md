# Pending-release validation checkpoint

Current source includes the TV queue-first Back and seek semantics improvements,
Wiki offline retained-image budgets and early download stopping, and matching
persistent-state write/read size bounds on the server. The published Android
APK remains 0.16.72; these latest Android changes are not yet packaged.

Verified in the combined run:

- Direct Android: 604 unit tests, no failures, errors, or skips.
- Store Android: 604 unit tests, no failures, errors, or skips.
- Frontend: 445 tests, no failures or skips.
- Complete Go suite: passed.
- Store-readiness and module-boundary checks: passed. Packaged checks inspect
  the existing APKs, not a newly built artifact for these source changes.
- Android lint: Direct and Store passed, each with zero errors and 84 warnings.
  The combined Gradle run completed in 5m 9s. Warning categories: 74 UseKtx,
  six dependency/update notices, two logging suggestions, and two UsableSpace
  suggestions. No automatic dependency upgrades or commit/apply rewrites were
  made merely to silence warnings.

Physical TV, TalkBack, and device-specific lifecycle behavior remain unverified
without connected hardware. Unit coverage and compilation are not substitutes
for those checks. No APK publication occurred in this checkpoint. Server-only
persistence fixes were subsequently deployed at 16:13 UTC; see
`STATE_WRITE_BOUNDS_2026-09-07.md` for deployment evidence.
