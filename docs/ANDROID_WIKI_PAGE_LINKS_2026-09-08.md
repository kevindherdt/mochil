# Native wiki page link context

Pending after 0.16.78; not packaged or deployed.

Page matching moved out of general wiki models into `WikiPageLinks.kt`. The ViewModel supplies the selected page path. Matching now prefers the local folder, supports explicit dot-relative paths within the vault, then considers root paths and unique fallback labels. Exact case remains usable; ambiguous case-folded paths or labels do not pick the first scanned note. Absolute targets, schemes, control characters and above-root traversal are rejected.

Both Android debug suites and module checks passed, including order reversal, local/root priority, relative boundaries, ambiguous names and case-sensitive paths. The existing missing-link workflow remains separate and does not automatically create files. This retains the current native `.md` page boundary; `.markdown` support and shared indexed lookup across all clients remain follow-ups. No performance improvement is claimed for the still per-click page search.

## Allocation follow-up

Per-click lookup no longer creates page-wide filtered lists, matching lists, distinct-ID sets, or extension-stripped path strings. A direct scan retains one matching entry, terminates on a second distinct identity, and uses bounded region comparisons against original paths. Exact/case-insensitive and relative/root priorities remain unchanged. A 10,000-page test covers exact, folded and label lookup with repeated versus distinct identities.

Both full Android debug suites and module checks passed (11 seconds). This is a code-level allocation reduction; no measured device latency or RAM percentage is claimed, and lookup remains linear rather than introducing a persistent catalogue index. No build/deployment beyond unit-test compilation.
