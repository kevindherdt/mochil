# Playlist refresh ordering

Status: deployed in the web/server-only update at 15:24 UTC on September 8, 2026.

Web favorites already isolate per-track writes and reconcile toggles completed
during reads. The parallel playlist overview path instead allowed an external
refresh to start while a queued write was running. That snapshot could replace
the locally updated overview after the write completed.

External overview refreshes now wait for the current write queue tail and recheck
it after awaiting, including edits queued during the wait. Internal reconciliation
inside an edit remains direct, avoiding self-deadlock. Existing revision checks
continue to reject stale reads started before a new mutation begins.

Regression tests cover refresh during two queued creations and refresh following
a rejected write. All 563 frontend tests, module-size checks, desktop/mobile
playlist form smoke checks and delayed-save playlist drag checks passed. No layout,
API contract, server data or APK changed in this follow-up.
