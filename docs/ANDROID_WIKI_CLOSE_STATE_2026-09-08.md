# Wiki close during loading

Status: source change after 0.16.84; pending a later APK batch.

Closing a page previously cancelled its request but retained `loading = true`.
The cancelled response would not clear that flag, leaving a stale loading state.
Page close now explicitly ends loading and clears page-specific missing-link,
attachment status, draft and navigation state. It also cancels any stale tree
refresh already invalidated by the page-session reset.

The vault tree, expansion, search and independent offline mirror state remain
intact. Tests cover closing while loading, unrelated state preservation and
idempotence. Both full Android debug unit suites passed in 14 seconds; module
gate passed. Log: `/tmp/bobobooks-wiki-close-tests.log`.

No control layout changes. Physical-device spinner behavior remains unverified;
the state transition and request cancellation were inspected/tested in source.
