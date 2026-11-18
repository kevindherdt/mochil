# Post-0.16.80 reliability batch

This batch shipped as 0.16.81/code242 on 2026-09-08. See `ANDROID_RELEASE_0.16.81_PREPARATION.md` for authoritative artifact/deployment evidence. Subsequent sections retain chronological preparation notes.

- Web audio/video: retain newer seek intent across late responses, conflict resolution and network failures. Video progress responsibility is separated from player controls.
- Android audio outbox: skip unchanged replay-read persistence, serialize overlapping instances with a shared lock, preserve unreadable data, and record a content-free failure diagnostic.
- Android video outbox: preserve unreadable encrypted/JSON data rather than treating it as an empty journal.

Evidence: 548 frontend tests and module checks passed for the web changes. Android targeted tests cover overlapping journal instances, stale acknowledgements, failed encrypted reads and malformed/oversized stored data. Full post-video-change Direct/Store unit run: session 21521, `/tmp/bobobooks-post80-journal-full-tests.log`.

Before release: refresh full Go/web/browser gates and Android lint, then build/version/sign/verify a substantial batch. Do not use old APKs as evidence for current source. UI layout is intentionally unchanged. Physical Android TV/mobile acceptance, broader wiki features and the full roadmap remain open.

## Integrated verification follow-up

- Full Go tests, vet, module gate and source/distribution gate passed (session 4202).
- Playlist form fixtures at 1440/390px, desktop playlist drag ordering, and wiki rendering/task recovery fixtures at 1440/390px passed (session 8292).
- New `npm run test:browser:video-progress` passed at 1440/390px (session 45444). It imports the real player and exercises DOM seeking/timeupdate/close/pause bindings with a synthetic clock and transport. An offline seek retries with explicit intent; close issues one final checkpoint and later pause cannot duplicate it. This is not codec or visual acceptance.
- Android lint remains running in session 13460, log `/tmp/bobobooks-post80-lint.log`. Poll the existing process; do not relaunch based on elapsed time.

No new APK or deployment in this follow-up.

## Playlist creation safety

The HTTP collection POST used whole-playlist replacement with an empty list. A repeated create with an existing name could erase tracks. A regression reproduced the old 201 response; creation now uses an atomic create-only store method and returns 409 for duplicate names, including case/whitespace equivalents. Stored title/order/tracks remain unchanged. Account scoping and persistence-failure behavior are tested, including concurrent creates under Go's race detector. Full audio/web package tests passed (`/tmp/bobobooks-playlist-create-tests.log`).

Cross-device replacement edits still need a revision/conditional-write protocol across both web and Android; this create fix does not solve that separate issue.

Android lint session 13460 subsequently completed successfully; `/tmp/bobobooks-post80-lint.log` is authoritative. Source remains unshipped after 0.16.80.
