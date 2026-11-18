# Wiki structural mutation safety

Status: worktree changes after published 0.16.75; not packaged.

Inspection found that page rename/move and trash clear the recovery draft before making the request, and successful page creation replaces the selected page. The existing page menu permits these actions during editing. A shared model policy now refuses these page-changing operations while the current editable draft differs from the saved content, reporting save/cancel guidance through the existing error surface. The check occurs before recovery cancellation/clearing or remote requests. Intentional deletion to empty text is still an unsaved edit. Creating a folder remains allowed because it retains the page and draft.

Both full Direct and Store unit-test tasks passed. Policy tests cover changed text, empty edited text, unchanged editing, and ordinary browsing. No physical-device interaction test was performed and no APK was rebuilt.

Architecture follow-up: the WikiViewModel still coordinates reading, draft persistence, offline caching, and structural mutations. The next useful extraction is the structural mutation request lifecycle with injected repository/state callbacks and race tests—not an arbitrary line-count split. Existing page-session ownership and recovery intent ordering must survive that extraction. Native revision-history support remains separate outstanding work.

## Create-result ownership

Creation reconciliation now lives in `WikiCreateState`, separate from Android lifecycle and HTTP. A delayed page-create response updates the tree but leaves the current page selected when its draft changed in flight. Folder creation always retains the existing editor state and draft; previously it set `editing=false`, hiding ongoing edits even though the folder operation itself should be harmless. Unchanged page creation still opens the new page normally. Tests cover these transitions plus stale-library identity. The existing page-session guard remains in the caller.

Move/trash in-flight editing behavior remains a follow-up; this extraction does not claim to finish all structural mutation ownership work. Changes remain unbundled after 0.16.75.

## Move-result ownership

Move reconciliation now has its own pure state transition. New text (including an intentional empty draft) typed during the move remains editable, but is rebound to the moved page identity/path and acknowledged baseline. The ViewModel then reschedules recovery persistence using that new identity, superseding pending old-identity writes through the existing write-ticket gate. An unchanged move still closes edit mode normally. Tests cover both outcomes and stale library/page/path identity.

Trash completion with in-flight typing remains a separate follow-up. No APK or deployed server changes accompany this work.

## Confirmed trash ownership

After the existing dirty-draft guard and explicit user confirmation, trash now exits editing while the request runs. Queued editor callbacks cannot change the draft during a pending non-editing mutation. The underlying saved page remains visible until success; on failure it stays available for reopening the editor. Ordinary saves keep `editing=true`, so their established newer-text preservation remains supported.

Policy tests verify trash retains the prior page/text while rejecting pending input, ordinary saves continue accepting input, and completed mutations unlock draft reconciliation. This is not a new confirmation dialog or a redesign. Full Direct and Store unit-test verification accompanies the change; physical-device behavior remains unverified. All structural-mutation changes in this document are still after the published 0.16.75 build.

## Acknowledged writes versus failed refreshes

Create-page, create-folder, move, and trash now distinguish successful server writes from failures in the subsequent tree/page refresh. The repository wraps only the post-acknowledgement read phase in `afterWikiMutation`; mutation failures are not mislabeled as success. A completed operation followed by a failed refresh reports what actually happened and asks for a vault refresh, without automatically repeating the write. Cancellation remains cancellation, and each dependent read checks coroutine ownership. A created page absent from the refreshed index is also reported as incomplete reconciliation rather than silently treated as normal success.

Repository tests cover all four operations with a failing tree refresh, ordinary folder creation without a selected page, and cancellation identity. This improves reporting but does not yet implement automatic reconciliation of a moved identity after a failed reload; current recovery text is retained locally. No release was published for these changes.
# Backlink request ownership

Post-0.16.75: backlink loading no longer converts cancellation or network failures into an empty successful result. Successful results are checked against the initiating page session as well as the existing library/page identity reducer. This prevents a cancelled request from clearing backlinks after navigating away and back to the same page. Genuine successful empty responses still clear obsolete backlinks; failed refreshes preserve the previous result.

Coroutine tests cover successful empty delivery, network failure, explicit cancellation propagation, and a loader returning after its coroutine is cancelled. Both Android unit-test suites and module checks passed. The first test compile used an unavailable coroutine-test API; tests were corrected to the existing runBlocking setup without adding dependencies. No APK was rebuilt.
