# Android lint triage after 0.16.77

The published candidate's Direct and Store debug reports each contain zero errors and 85 warnings. The Direct report groups as follows:

| Category | Count | Assessment |
| --- | ---: | --- |
| UseKtx | 74 | API/style suggestions; avoid mechanical changes that accidentally replace synchronous durability with asynchronous preference writes. |
| Dependency/version suggestions | 6 | Review separately with compatibility and release notes; lint availability notices alone do not justify immediate upgrades. |
| LogNotTimber | 3 | Logger consistency suggestions, including the new sanitized playback delivery warning. No Timber setup was found in app sources. This accounts for the extra warning compared with the prior release total. |
| UsableSpace | 2 | Suggests allocatable-storage APIs; current conservative free-space check is not itself a correctness defect. |

## Concrete storage boundary fix

Inspection of the free-space policy found `expectedBytes + reserve` could overflow with an extreme reported size and incorrectly pass the reserve check. Validation now rejects insufficient/invalid available space first, then compares the expected size with `availableBytes - reserve`. Existing maximum-file-size rejection remains first.

Tests cover Long.MAX_VALUE, exact fitting/reserve boundaries, unknown size, and invalid free-space reports. Both full Android unit suites passed (703 tests per variant), and module checks passed. This edge-case fix is not in the published 0.16.77 APK; retain it for the next batch. No UI, dependency, or deployment changes were made.

This is lint triage plus one inspected policy fix, not a complete security/performance audit or proof that every warning is harmless.
