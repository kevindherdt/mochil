# Native vault attachment gap and missing-note safety

Status: audit plus source-only validation fix after 0.16.79.

Native `WikiClient.asset` intentionally accepts only image MIME responses with a 3 MiB cap; its current offline pipeline cannot be advertised as arbitrary attachment mirroring. `WikiViewModel.openLink` resolves Markdown pages and otherwise enters missing-note creation. A PDF link is not currently routed to the native PDF reader.

The missing-note helper previously stripped leading/trailing slashes after link parsing. An absolute reference such as `/Private/Note` therefore became a proposed local note. It now rejects absolute/slash-normalized paths, colon-bearing URI/drive-style references, controls, empty path segments and traversal before offering creation. Valid Unicode note names, aliases, headings and ordinary dotted titles retain their behavior. This is input hardening, not evidence of an existing server filesystem escape; server validation remains independently required.

Direct and Store focused missing-link/model tests passed (9s Gradle execution). No attachment download capability, new APK, deployment or personal file mutation is claimed.

For actual attachment support, add a dedicated indexed-attachment resolver and bounded authenticated download/open owner. Reuse native PDF opening where possible; external-app handoff must use a scoped content URI, never expose the server token or private file path. Define format and byte limits, cancellation/partial-file cleanup, storage quotas and offline revision invalidation separately from the small image cache. Preserve the existing note tree and do not render arbitrary attachments as editable Markdown pages.

## Indexed-file foundation

Extracted path/identity matching into `WikiFileIndex`, parameterized by an explicit format allowlist. Existing `WikiAssetIndex` now delegates with the same image-only policy; it does not start accepting PDFs as image embeds. Entries without an indexed identity are refused. PDF-policy tests cover local/root precedence, exact case, ambiguous folded matches, bounded traversal and scheme rejection. This is not yet wired as an attachment opener.

The existing `PublicationOpener.prepareFile` can route a validated app-private PDF to `PdfReaderActivity`. That activity accepts only canonical files under `filesDir/publications`; use a dedicated bounded attachment subdirectory rather than widening the activity's path boundary. A `local:` progress key avoids misrouting vault attachment progress through publication APIs, which do not serve vault item identities. Preserve page context on return.

## PDF download owner (not yet connected to UI)

`WikiPdfDownloads` now owns a separate, disposable four-file PDF cache with a 64 MiB per-file cap. It streams authenticated document-asset responses through a 32 KiB buffer, checks MIME/header/announced and actual length, reserves storage before transfer, and atomically publishes only complete files. Cache filenames hash server/device/vault/item/revision identity; Markdown cannot supply filesystem paths or an arbitrary fetch URL. Only generated cache names are eligible for eviction. Revisionless entries are redownloaded rather than reused indefinitely.

Direct and Store focused tests passed (9s Gradle run), covering encoded endpoint construction, cache reuse/revision/account separation, invalid MIME/header, unknown-length oversize, truncated response, disconnect/failure cleanup and bounded owned-file eviction. Module gate passes. Coroutine cancellation checks and `finally` cleanup are implemented, but a dedicated cancellation regression is still needed. Process-death orphan-part cleanup and cross-instance cache coordination need review before UI wiring. A PDF header check is not full PDF validation; the native renderer remains responsible for parsing.

No PDF-link action is enabled yet. Next: finish cache lifecycle safeguards, then connect an independently owned opening job with loading/error feedback, stale-selection cancellation, and local-only PDF progress. This component does not advertise general attachment mirroring or full-vault offline support.

### Lifecycle follow-up

Download serialization now uses a process-wide mutex, so overlapping ViewModel/component instances cannot prune or replace one another's active temporary files. At preparation time, only this component's numeric generated `wiki-pdf-*.part` filenames are removed as process-death leftovers; unrelated `.part` files are preserved. The cancellation regression cancels the actual coroutine during streaming and verifies no published PDF/partial remains and the connection disconnects. The orphan cleanup regression verifies the filename boundary. Both Android flavors' downloader suites passed (9s Gradle run); module gate passed. UI/opening integration remains outstanding.

## Native opening integration (source, unpublished)

`WikiPdfOpening` owns the cancellable prepare/open lifecycle independently of the ViewModel. Indexed `[[Manual.pdf]]` links now prepare the bounded cache and use the existing native PDF reader through `PublicationOpener.prepareFile`. The progress identity is explicitly local-only and revision-scoped; vault attachment IDs are not sent to publication-progress APIs. Unknown PDFs produce an error rather than an offer to create `Manual.pdf.md`.

`WikiAttachmentLifecycle` cancels pending work on screen disposal or lifecycle stop. Selecting/closing a note, changing vaults, or following a different wiki link also cancels it. A small theme-token status line uses the existing note layout; it does not replace the page or add a modal. Returning from the PDF keeps the note state. Physical navigation/rendering acceptance is still unavailable.

Relative Markdown `[label](file.pdf)` and `[label](Note.md#Heading)` links and local heading links now route through the vault callback too, preserving literal `+` and decoding escaped paths once. External HTTP(S) handling remains separate. Unsafe/unsupported destinations remain inert. A compatibility test caught malformed-link greediness during implementation; the corrected pattern passes the existing differential formatting oracle.

Both complete Android flavor unit suites executed successfully: 784 tests each, zero failures/errors/skips. The opt-in benchmark test is included but does no timed work without its environment switch. Module gate passed. Combined lint remains in progress in session 67244, log `/tmp/bobobooks-wiki-pdf-integrated-fixed.log`; do not claim completion until observed. No APK/version bump or server deployment occurred. Impeccable hardening guidance kept the UI change to existing layout/status conventions.

Limits: only PDF attachment opening is connected; arbitrary attachments and full offline vault mirroring are not. PDF page-number fragments are not applied yet. Cache entries may be evicted and do not appear as explicit user downloads. Remote content changes depend on the refreshed indexed revision, not continuous background monitoring.

### Server route verification

`TestAndroidVaultPDFAttachmentRequiresDeviceAndServesExactBytes` now exercises the Android API router, real device-token store and real temporary vault file together. Exact PDF body/MIME/length, `nosniff`, HEAD, partial range, unknown identity and missing/revoked credentials passed (`go test ./internal/web -run TestAndroidVaultPDFAttachment -count=1`). This verifies the existing server route needed by the native downloader, without personal credentials or media. It is not a physical-device end-to-end reader test.

Direct lint completed with zero errors and 85 warnings. Store lint was last observed at `lintAnalyzeStoreDebug` in the same session 67244; retain that process until its terminal result is observed.

Final observation: session 67244 exited zero. Both Android flavors' complete unit tests and debug lint passed; each lint report has zero errors and 85 warnings. No active build remains from this verification. Warnings and physical reader/lifecycle acceptance remain outstanding, and this source batch is unpublished.
