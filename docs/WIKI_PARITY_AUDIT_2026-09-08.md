# File-backed wiki parity audit

Scope: code inspection, not physical-device or full rendered acceptance. The parser/storage and mutation-recovery batch below was published in 0.16.78; chronological unpublished notes describe its development. Later follow-ups explicitly identify work pending the next batch.

## Systems that must not be conflated

- `internal/web/wiki_http.go` serves the legacy account-scoped managed store through `/api/wiki`, including history, restore, and restore-version routes.
- `internal/web/vault_http.go` serves file-backed vault mutations: page/folder creation, page update, move, and trash. It exposes no revision-history or restoration operation.
- Android `wiki/WikiClient.kt` uses the vault API, not legacy managed history.
- Web `document-library.js` uses vault tree/create/editor modules and `vault-markdown.js` for the current document library experience.

Existing legacy history therefore does not establish file-backed vault history support. Retiring old surfaces must preserve user data and needs an explicit migration path, not blind deletion of the legacy store.

## Confirmed gaps and dependencies

Current status (0.16.80 candidate verification): items 2 and 4 below describe the original audit, not outstanding implementations. Structured mutation outcomes and client recovery were implemented in the follow-ups; web frontmatter, tables, task persistence and heading navigation now pass real-server browser fixtures at 1440px and 390px. Native PDF attachment opening is included in the candidate, but does not provide general attachment/offline-vault mirroring. Items 1, 3 and physical-device acceptance in 5 remain open. The chronological findings are retained for traceability.

1. File-backed history/restore: server capability is missing. Define bounded retention, storage ownership, restore conflict checking, external-edit handling, and recovery before adding client controls. Do not silently begin recording every private vault revision without a clear configuration/retention policy.
2. Web Markdown parity: the inspected shared renderer has headings, lists, quotes, fences, rules, links and inline formatting, but no native table/task-checkbox/frontmatter rendering equivalent to Android. Web wiki heading links also strip the fragment in `vault-markdown.js`; Android has an explicit pending heading-jump flow. These are functional parity tasks, not reasons to redesign the established shell.
3. Non-image attachment mirroring remains outside the native Markdown/image offline pipeline. Define format limits and offline opening before advertising full-vault offline availability.
4. Post-write index failure: vault mutation HTTP returns an error after the file was successfully changed if rescanning fails. Existing client helpers distinguish some successful writes from later reload failures, but this server-side partial-success case needs a structured outcome and regression coverage across clients.
5. Android TV and physical phone acceptance remain unverified in this environment. Automated state/parser tests cannot prove remote focus, readability, or on-device rendering performance.

## Next implementation order

First resolve structured vault write/index outcomes, since ambiguous success can prompt a duplicate create/retry. Then improve web Markdown/heading navigation parity while preserving current layout and safe escaping. History/restore is a larger server-first design task with retention choices; it must not be mislabeled as a small Android-only feature.

No source behavior or live deployment was changed by this audit. Broad app goals remain active.

## Follow-up: vault mutation outcome contract (unpublished)

The server now distinguishes a completed filesystem mutation from an index-refresh
failure. HTTP 500 is retained for compatibility, but the body explicitly includes
`code: "vault_index_refresh_failed"`, `committed: true`, and
`index_refreshed: false`, with a warning not to repeat the completed operation.
Normal 200/201 responses include `ok: true`, `committed: true`, and
`index_refreshed: true`. Genuine write failures do not claim a commit.

`vault_mutation_outcome_test.go` uses real temporary vaults and a directory at the
cache-file destination to force index persistence failure. Coverage checks the
actual filesystem contents for create, update, move, trash, and folder creation,
successful index outcomes, and a conflicting repeated create. Targeted vault
tests pass. This is not yet a complete client fix: the web JSON helper currently
discards the structured fields, and Android throws a message-only WikiException.
Both clients still need explicit committed-outcome handling so retries are not
offered as if the write itself failed. No live deployment was made.

### Web create/editor follow-up (unpublished)

The shared JSON helper now preserves bounded error codes and the mutation outcome
fields. A small vault-specific predicate requires the exact 500/code/boolean
contract; ambiguous network errors do not count as committed writes. Create-page,
create-folder, and editor forms keep submitted text read-only and submission
disabled after an explicit committed/index-failed response, retaining the existing
status area and changing Cancel to Close. Text remains available for selection
and copying. Ordinary failures remain retryable.

The editor also guards the separate successful-write/failed-reload callback case,
which previously re-enabled Save with a stale revision. Browser interaction tests
pass at 1440px and 390px for both failure paths, duplicate-submission prevention,
and dismissal. These fixtures establish behavior, not whole-app visual acceptance.
Android and any other vault mutation surfaces still require their own recovery
handling; this does not establish complete cross-client resolution.

### Android response/repository follow-up (unpublished)

`WikiResponseError.kt` now owns typed API error decoding. Only the exact HTTP
500/code/boolean outcome acknowledges a committed mutation; malformed JSON, HTML,
legacy messages, and string booleans cannot do so. WikiClient uses this decoder
for its already byte-bounded error responses. The save repository converts that
acknowledgement into its existing saved-but-reload-failed outcome without issuing
another request. This preserves committed task-checkbox content instead of rolling
it back as if the write failed. Cancellation and unacknowledged errors retain
their existing semantics.

Direct and Store debug unit suites pass, including new decoder and repository
regressions. No APK or deployment was produced. Remaining: explicit Android
mutation lock/recovery state, including create/move/trash and successful-write but
failed-follow-up-read cases. Current ViewModel handlers still clear `saving` after
those errors, so preserving the acknowledgement alone is not a complete retry
prevention fix.

### Android session-local save barrier (unpublished)

The repository now records acknowledged-save/reload failures in `WikiSaveRecovery`,
keyed by server, device, vault and page. Subsequent save, move and trash calls for
that page are rejected before network IO until a successful authoritative page
read clears the barrier. A tree-only refresh does not clear it. Read tickets
prevent responses started before the failure, or before a newer failure, from
clearing the current barrier. Other page identities remain usable.

Both Android debug unit suites and focused recovery tests pass. The existing
error presentation continues to request a refresh; no visual layout was changed.
The barrier is session-local, not persisted across process death, and is not a
general mutation serializer. Follow-up remains: recovery from acknowledged
create/move/trash failures themselves, and reflecting blocked mutation state in
controls without disabling browsing or losing newer draft text. A dirty-editor
tree-only refresh intentionally cannot unlock writes; reopening the page obtains
the authoritative revision through the existing page-load flow.

### Android structural retry barrier (unpublished)

`WikiStructureRecovery` now wraps create-page/folder, move, and trash operations.
An explicit committed server error or failed post-write reload records the
expected tree change and prevents repeating the operation. Page identity keys
also block saving/trashing a page with an unresolved move, or moving/saving one
with an unresolved trash. Other vaults and unrelated identities remain usable.

A subsequent tree read clears only matching recovery tickets: creation requires
the requested path and kind; move requires the destination plus absence of the
old ID; trash requires absence of the old ID. Older in-flight reads cannot clear
newer acknowledgements. If indexing is still broken, a stale tree cannot unlock
the operation; the index must first be repaired/rescanned. Error wording no longer
invites repeating a completed operation.

Unit coverage includes all four structural operations, both server-side and
follow-up-read failures, stale/older listings, identity separation, typed versus
unacknowledged errors, and repository-level duplicate-request prevention. The
barriers remain session-local, not general concurrency serialization or durable
operation journaling. Control-level disabled/recovery presentation and physical
device acceptance remain outstanding. No build artifact or deployment changed.

### Cancellation audit follow-up (unpublished)

A successful write followed by cancellation previously bypassed recovery recording.
Both save and structural pipelines now retain their acknowledged-write barriers
when cancellation interrupts the follow-up phase, while propagating the original
cancellation unchanged. Cancellation before write acknowledgement still does not
claim success. Regression tests cover cancellation immediately after the write
(no follow-up IO), during refresh, and subsequent recovery through a fresh read.
This closes a session-local retry gap; it does not make acknowledgements durable
across process death or resolve requests whose HTTP outcome was never received.

### Request setup cleanup (unpublished)

Wiki mutation setup previously returned a connection to the response reader only
after writing its body. Failures opening, writing, or closing the output stream
therefore bypassed the response reader's disconnect. `WikiMutationConnection.kt`
now owns cleanup until setup succeeds and transfers ownership afterward. Cleanup
failures are suppressed onto the original exception rather than masking it.

Tests inject setup/open/write/close failures, verify successful UTF-8 output,
existing timeouts, redirects remaining disabled, and bodyless setup without an
output stream. Both Android debug unit suites pass. This is resource-lifecycle
hardening, not a measured CPU/RAM reduction or an automatic retry policy; an
interrupted upload can still have an unknown server outcome. No APK/deployment.
