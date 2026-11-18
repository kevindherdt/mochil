# Web vault request ownership

Status: post-0.16.75 worktree, not deployed.

Resetting a document/vault library now cancels and invalidates any page-text request. Leaving the surface invalidates both request identities, not just their abort signals. Late errors from old text, forward-list, and previous-list requests cannot replace the current view/status. Stale previous-list finalizers cannot clear another request's loading flags.

Controller tests execute the actual document-library controller with mocked mounting/DOM dependencies and deferred requests that deliberately ignore cancellation. They cover stale library failures, late success after abort, and late page content after switching libraries. All three pass. These are controller behavior tests, not rendered browser acceptance.

The audit also confirmed a capability gap: the web vault currently converts Obsidian embeds into attachment links rather than displaying inline local images as Android does. A dedicated safe attachment resolver and renderer remains needed; this change does not claim image feature parity.

## Inline image foundation

Added `vault-assets.js`, a standalone indexed resolver for the upcoming renderer integration. It matches relative/root image paths, preserves shorthand lookup, rejects broken explicit paths instead of substituting unrelated filenames, restricts entries to the selected library and the server's raster-image formats, and builds authenticated same-origin document routes rather than trusting Markdown URLs or metadata open_url fields. Four focused tests pass, including hostile schemes, cross-library entries, SVG rejection, and URL escaping.

This module is not wired into the rendered page yet. Next: renderer support that respects fenced/inline code, uses escaped alt text and lazy images, preserves attachment-link fallback, and fits the incumbent reading layout. No visible feature or deployment is claimed for the resolver alone.

## Renderer integration completed in worktree

The subsequent integration now connects `createVaultMarkdownRenderer` to opened vault pages. Obsidian `![[image.png]]` embeds resolve through the indexed authenticated routes and render lazy, asynchronously decoded images constrained to the reading pane. Numeric width suffixes are bounded and remain responsive. Missing/unsupported attachments retain link fallback. Inline/fenced code is preserved instead of undergoing the previous global wiki-link replacement.

The shared Markdown renderer has an optional trusted inline-text rendering hook; ordinary document rendering retains its escaped default. The vault-specific implementation owns resolution and escaped HTML output separately from the library controller. Standard Markdown image syntax and full CommonMark/Obsidian compatibility are not claimed by this increment.

All 471 frontend tests and module checks passed. An isolated Chromium fixture using the real renderer and vault CSS passed image loading and no-overflow assertions at 1440px and 390px; screenshots were inspected at both sizes. This verifies embed sizing and literal code, not the whole themed app or authenticated production delivery. Impeccable hardening guided responsive containment and preserving code/unsafe-input behavior. No deployment occurred.

## Standard Markdown image syntax

The next increment adds basic inline `![description](path)` images, including optional quoted titles and URL-encoded paths, through the same resolver and image markup. Explicit empty alt text stays empty. Malformed encodings, external schemes, and unresolved paths remain escaped source text instead of loading images. Fenced and inline code remain literal. Complex balanced-parenthesis destinations, angle-bracket paths, reference-style images, and a full CommonMark grammar remain outside this parser's current coverage.

All 473 frontend tests and module checks pass. A Chromium DOM check confirms escaped alt text does not create elements and remote image syntax does not produce a remote image request. The responsive image CSS is unchanged from the prior two-width check. Impeccable hardening informed safe fallback and preserving the existing layout. No deployment occurred.

## Page tree versus attachment catalogue

Added an independent page projection for navigation: folders and Markdown pages stay in the web sidebar, while raster images/PDFs/other attachments remain in the complete catalogue used for resolution. Sidebar search and counts use the same projection. The tree renderer also applies the projection defensively. Input catalogue arrays are not mutated, so embedded image lookup retains all attachments. A regression test covers Markdown extensions, folders, hidden attachment rows, and source-array preservation. Full frontend tests and module checks passed. This is not a general-purpose attachment-management UI.

## Empty-vault creation remains reachable

The workspace now remains mounted when there are no visible pages. This preserves the existing sidebar toggle and authorized create-page/folder actions in a new vault or an empty search result. Empty vaults say “No pages yet”; unmatched searches retain their existing message. Read-only vaults never receive create controls. Two controller regressions cover these cases; full frontend tests and module checks passed. Impeccable empty-state guidance informed preserving the existing controls rather than introducing a separate onboarding UI. This change is not deployed.

## Create/save form ownership

Create and edit handlers now reject repeated submission while a write is pending. Inputs are temporarily read-only (format buttons disabled for the editor), preventing a successful save/reload from discarding text entered after submission. Failure restores editing with the draft intact. Detached forms cannot invoke completion navigation or update a later page. Cancelling the UI does not undo a write already accepted by the server, and these changes do not claim otherwise.

`node scripts/browser-vault-mutation-smoke.mjs` passed at 1440px and 390px using real form modules and deferred requests. Checks cover duplicate submission, pending read-only state, failed-save draft retention, restored editing, and cancellation followed by late success without navigation. No browser page errors occurred. The changes preserve the existing form layout; Impeccable hardening guided request ownership and input preservation. Not deployed.
