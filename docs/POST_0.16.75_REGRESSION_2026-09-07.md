# Post-0.16.75 regression checkpoint

Status: source changes tested, not packaged or deployed. The live Mochil container remains running with zero restarts at this checkpoint.

## Verified

- `go test ./...`: passed (unchanged packages reused cached results).
- `go vet ./...`: passed.
- `npm run test:frontend`: 461 passed, no failures or skips.
- `npm run test:browser:playlists`: passed at 1440px and 390px using the isolated browser fixture and real feature modules. Covers playlist loading, failed rename/input retention, stale completions, and queue retry behavior; it is not a full-site visual assessment.
- Both Android debug unit-test tasks: successful, latest rerun up-to-date. Current XML results contain 663 tests each for Direct and Store, zero failures/errors. The preceding implementation turns executed changed tests.
- Store-readiness script: passed. Its existing APK inspection covers previously built artifacts, **not** the unbundled source changes.
- Module checks passed in the implementation turns.
- Race-enabled vaultwriter and web tests passed after the vault durability changes.

## Accumulated changes awaiting packaging

- Android wiki mutation guards, retained draft handling, and clear post-write refresh failures.
- TV music queue persistence scope and identity-based actions/focus target selection.
- Playback queue duplicate batching, metadata retention cleanup, and preparation-before-mutation ordering.
- Bounded vault revision reads and destination-first durability ordering for move/trash.

See the linked topic reports in this directory for exact limitations and tests; this checkpoint does not claim completion of the broader roadmap.

## Outstanding verification

No physical Android TV remote, TalkBack, or mobile playback acceptance was performed. Release optimization/signing/package inspection and deployment have not been rerun for this batch. Filesystem hardening still needs a separate review of external path-replacement races; the durability change is not a multi-directory atomic transaction. Active queue capacity beyond the initial 200-item window still needs a user-intent-preserving policy. Native wiki history and other roadmap capabilities remain unfinished.
