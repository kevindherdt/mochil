# Android wiki recovery boundary audit

Status: shipped in Android 0.16.75, published September 7 at 18:47 UTC. Chronological pending-packaging notes below describe the preparation stage.

The native wiki roadmap and implementation already include editing, revision-checked saves, encrypted drafts, offline pages, and folder navigation. Native revision history remains a larger outstanding feature.

## Exact recovery

`WikiDraftStore.load` previously silently shortened recovered content to two million characters. It now validates complete content and page/library identity before returning a recovery draft. Invalid content is rejected rather than converted into a partial draft; decoding itself does not mutate or clear stored content. Existing owner-mismatch isolation is unchanged.

Draft saves previously returned normally when identity or size validation rejected a write, making the persistence wrapper report success. Invalid writes now throw through the existing persistence-failure path. Superseded write tickets still intentionally do nothing, preserving intent-order protection.

Tests cover exact CRLF/Unicode recovery, intentional empty content, the size boundary, oversized-content rejection without truncation, and missing/blank identities. Both full Direct and Store debug unit-test tasks and the module gate passed. No rendered UI or physical-device recovery test was performed, and no personal vault files were modified.

Remaining limitation: invalid stored draft decoding currently returns no recovery draft through the existing nullable API. An explicit damaged-draft recovery/export flow is not implemented. The editor's separate input-length behavior was not changed in this storage-layer audit.

## Editor follow-up

The editor and ViewModel also silently truncated oversized input. Both now reject the complete oversized edit without changing the existing document. The editor retains its previous selection and IME composition; typing, paste, and all formatting actions share the same acceptance function. The ViewModel rejects before cancelling or replacing the prior draft's pending persistence, and reports a specific error through the existing page error surface. A subsequent accepted edit clears only that limit error, not unrelated errors.

Tests cover oversized paste, bold/heading/fenced-code expansion at the boundary, exact maximum-sized input, and intentional deletion to empty text. Impeccable hardening guidance informed reuse of the existing error surface without redesigning the editor. Physical paste/IME and rendered error-state verification remain pending; no attached Android device is available. No APK was rebuilt.

## Remote page validation

Save lifecycle inspection confirmed existing newer-draft preservation and page-session ownership. Page response parsing, however, defaulted missing content to an empty string. The dedicated page decoder now requires exact requested identity and actual text content, checks the existing size bound, and requires a nonblank edit revision whenever permission is writable. Missing permission remains read-only; string-coerced permissions and malformed revisions fail closed. An explicitly empty page is still valid. The Go document endpoint was inspected and emits the required fields for writable Markdown vault pages.

New decoder tests cover empty-page success, missing/non-text content, identity mismatch, malformed permissions, missing revision, and read-only compatibility. Full Direct and Store unit-test tasks and module checks passed. No server API or vault data was changed.

## BOM rendering and task targeting

The server's exact-text round-trip fix preserves a leading UTF-8 BOM. Android now ignores that marker for heading/frontmatter presentation without stripping it from editable source. Task targeting understands BOM-prefixed frontmatter and skips fenced-code examples, matching the reader's backtick-fence handling. Previously a checkbox example inside a fence could consume a task ordinal and cause a real checkbox action to modify the example instead.

Regression tests exercise BOM/frontmatter, CRLF, fenced checkbox examples, first-line headings/tasks, and out-of-range task ordinals. Both complete Android unit suites passed. Targeted checkbox edits preserve every other source character. This does not add new Markdown syntax support; physical-device verification and packaging remain pending.
