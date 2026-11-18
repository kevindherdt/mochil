# Web music queue request ordering

Status: deployed September 7, 2026 at 17:57 UTC as a server/web update. Android remains 0.16.74. Earlier subsection deployment labels record the implementation sequence; publication evidence below is authoritative.

## Confirmed failure paths

The previous web queue save error callback reloaded the server queue unconditionally. An older failed save could therefore trigger a GET that overwrote newer local edits. The initial lazy queue GET also assigned its result unconditionally even when playback explicitly installed a new queue while that read was pending.

## Changes

- Queue edits advance a local request generation. Initial reads publish only if no newer edit occurred.
- Serialized writes carry detached entry identities and their generation. Older failures cannot modify current error state or trigger a stale reload.
- Failed latest saves retain local edits and expose a save error in the existing Queue destination, with an explicit retry action. Successful latest saves clear that error.
- No automatic full-queue read runs after a failed write.

Impeccable hardening guidance informed preserving edits and using contextual error feedback without changing the normal layout. The retry control exists only for a failed queue save.

## Verification and limits

Three new unit tests cover delayed initial reads, older save failure with a newer pending edit, and latest save failure plus explicit retry. Full frontend suite: 459 passing tests before the final import cache-version update. Module-size gate passes.

These are per-client request ordering guarantees, not cross-device compare-and-swap. No durable offline queue outbox was added: preserving edits in this session does not promise persistence through closing/reloading the browser. Initial queue-load failures still require a separate review, as do error visibility in the expanded player's queue panel versus the library Queue destination. This change is not a claim that all queue synchronization cases are solved.

## Initial-read follow-up (not deployed)

A failed initial GET previously marked the queue loaded and replaced it with an empty array. A subsequent append could consequently PUT only the new song, discarding the saved queue. Queue loading now distinguishes an unavailable queue from a confirmed empty array. Failed/malformed reads do not enable append/play-next writes. Explicit queue replacement from playback remains a deliberate operation and invalidates an older pending read.

Retry is state-aware: after a failed read it performs GET, not PUT; after a failed save it retries the retained queue. Queue destination feedback now uses the shared label “Retry queue sync”. Added tests verify zero writes after failed or malformed initial reads and preservation of existing tracks when an append is retried after recovery. Full frontend suite: 461 passing tests. Expanded-player error visibility and cross-device revisions remain outstanding.

## Expanded-player feedback (not deployed)

The expanded player's queue now reads the same synchronization error state and exposes the same retry action. It renders a contextual status message and button only during a failure, without adding a persistent toolbar control or changing normal queue layout. Error text is HTML-escaped. If retry removes the focused button, focus moves to Close instead of being lost. Existing track-identity focus preservation and bounded 60-row queue paging remain intact.

The queue-specific state binding lives beside the queue controller, keeping it out of the large library coordinator. Impeccable hardening guidance informed contextual feedback and focus recovery. Full frontend tests (461) and the extended isolated Chromium flows at 1440px/390px pass, including escaped error text, retry dispatch, disappearance after recovery, and focus fallback. These browser fixtures verify behavior, not full-app visual quality or physical-device accessibility. Module-size gate passes. Cross-device revision checks and durable offline persistence remain outside these fixes.

## Deployment verification

The web package tests, source/artifact readiness checks, and module-size gate passed before deployment. Image built as `opdshelf:web-queue-20260907`, then tagged local and deployed only the `opdshelf` service with `--no-build --no-deps`.

Live image: `sha256:6dca4b3eeac81f26e7c60ce3e5e3072c5de8a879e5408451faee46441ba8c2c4`. Container start: `2026-09-07T17:57:08.42451391Z`. Health is healthy, `/healthz` returns `ok`, zero restarts, and ten libraries loaded. Mounted Android APK remains 0.16.74 Direct with SHA-256 `1c90ee59a4a5d71be109edcbe11f72d995333a5a4864f313861fe5a6086f9fc7`; no new Android build was needed.

JavaScript responses use content ETags with `private, no-cache`, so a normal page reload revalidates updated modules rather than retaining immutable dependencies. Active already-open JavaScript contexts still require a reload to receive the changes.
