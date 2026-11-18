# Web wiki task controls: implementation progress

Pending after 0.16.78. Clickable task controls are now implemented in source; the follow-up below supersedes the initial groundwork notes. Not deployed.

The incremental Markdown line cursor now exposes each current line's original UTF-16 source offset. This lets the renderer associate a displayed checkbox with an exact source marker instead of searching repeated labels or counting task-looking examples in code/properties. Line text and LF/CRLF/CR handling remain unchanged; EOF reports source length.

Tests cover mixed line endings, optional BOM, Unicode/surrogate pairs, repeated task labels, code/property context and stable offsets across peek/take. All 528 frontend tests and module checks passed. A test initially referenced the wrong local variable and was corrected before the successful full run.

Remaining implementation: pass exact marker offsets through rendered task controls; preserve source/line endings while changing one marker; use existing revision-checked vault writes; guard completed-write/failed-refresh cases, duplicate clicks and navigation ownership; provide readonly behavior and accessible save/error feedback. This supporting cursor work does not establish those controls or write guarantees by itself.

## Interactive task follow-up

Unordered task-list items rendered by the vault now carry exact original marker offsets, including the frontmatter prefix. The dedicated `vault-task.js` module changes only that marker, preserves all other source characters, and uses the existing revision-checked PUT route. Readonly notes receive disabled controls. One pending operation disables task and Edit controls. Ordinary failures roll back the checkbox and permit retry; revision conflicts require reopening, and explicit committed/index-failed outcomes retain the checked state and block repeated edits. Detached or superseded pages are not updated by late callbacks.

Successful writes reload the authoritative page/revision, preserve pane scroll position, and restore task focus only when that reload still owns the page. The task row retains a 44px label target and inline status, without an additional toolbar. Impeccable guided native control semantics, compact feedback and focus handling.

All 530 frontend tests and module checks passed. `scripts/browser-wiki-task-smoke.mjs` (also `npm run test:browser:wiki-tasks`) passed at 1440px and 390px for single-marker CRLF writes, duplicate-click blocking, success, ordinary error, conflict, explicit committed failure and detached ownership. These mocked-network browser fixtures do not establish physical-device acceptance or a full live-server task save walkthrough. Nested Markdown grammar beyond the existing list parser remains limited. No APK or deployment.

## Server conflict contract

`vault_task_conflict_test.go` exercises real temporary files through the server handler. It loads a BOM/CRLF/Unicode task note, modifies the file externally without rescanning, and verifies that the old revision's task write returns conflict without changing the newer bytes. A fresh read exposes the external revision; a subsequent single-marker update succeeds and preserves both the other repeated task and external edit. The targeted test and module check passed. This verifies the HTTP/filesystem boundary separately from browser fixtures, not a combined live-browser walkthrough.

## Capability and post-save verification

Review confirmed that the current server writer, mutation routes, edit-revision response and web controls intentionally allow `.md` edits only; `.markdown` remains readable but is not writable in this batch. Extending that format requires coordinated server/document/client capability work, not enabling a web button alone.

The browser task fixture now also forces `onSaved` to fail after a successful PUT. The checkbox retains its committed state, the reopen warning appears, controls stay disabled and a second dispatched change produces no extra request. Both desktop/mobile widths passed. No production behavior or deployment changed in this verification pass.
