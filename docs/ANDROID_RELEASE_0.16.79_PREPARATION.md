# Android 0.16.79 / code 240

Status: 0.16.79/code240 published on 2026-09-08 at 10:57 UTC. Earlier verification notes below record the sequence; final outcome follows.

## Published outcome

- Combined Android build succeeded in 12m49s, 91 tasks executed and 85 up to date. Both optimized variants identify as 0.16.79/code240 and retain the previous signing certificate.
- Store archive `artifacts/releases/bobobooks-0.16.79-store.apk`: SHA-256 `80a46a4bd0bcd86fd66282eae5ee353ca6bc974794d9183d4778841d32ee1bab`. Store artifact remains private verification, not production store signing/submission.
- Explicit two-APK `scripts/check-store-readiness.sh` gate exited zero.
- Only the Mochil Compose service was recreated, using the tested candidate image and immutable Direct APK archive. Health is healthy, restart count zero, mounted APK hash matches the authenticated staging download.
- All ten libraries became ready. The pre-existing malformed EPUB (missing `META-INF/container.xml`) remains the one light-novel scan warning; no personal media was edited.
- Rollback remains `opdshelf:android-0.16.78` plus its retained Direct/Store archives. Pulse was not changed.
- Disposable wiki/update smoke containers were stopped after verification; host synthetic fixture notes remain in the documented temporary directory. No personal data was involved.
- Physical Android mobile/TV acceptance remains outstanding. This release does not complete the broader roadmap.

## Candidate verification

- Candidate image `opdshelf:android-0.16.79` built successfully: `sha256:2441aab3d5830321f6eb36f7d6d079306fd9d39b751e8752e25ed6e04ae6a279`.
- Isolated wiki server `bobobooks-01679-wiki-smoke` on loopback port 18102 is healthy. It mounts only `/tmp/bobobooks-wiki-stage-6mecvx`, containing synthetic notes, not personal libraries.
- `node scripts/browser-wiki-server-smoke.mjs http://127.0.0.1:18102` passed at 1440px and 390px against the full candidate shell and real HTTP/file writer: task save/reload persistence, collapsed properties, table rendering, heading navigation/focus and no document overflow or uncaught page errors. This is functional coverage, not comprehensive visual or physical-device acceptance.
- Combined Android debug tests, lint and optimized Direct/Store packaging remain running in session 15979, log `/tmp/bobobooks-01679-packaging.log`. Last observed progress reached Store optimized resource/Java resource tasks. Do not restart merely because completion has not yet been observed.
- Direct APK packaging completed, verified version 0.16.79/code240 and existing certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. Immutable archive `artifacts/releases/bobobooks-0.16.79-direct.apk`: SHA-256 `fcc3caec91166eda187e6768a4c097e012ae8f5304fb7445c986f8f6ba416d20` (7,062,376 bytes).
- Authenticated device login, release metadata and complete APK download/hash comparison passed against isolated `bobobooks-01679-update-smoke` on loopback port 18103 using synthetic credentials only.
- Both debug unit-test tasks executed successfully: 766 tests each, zero failures/errors/skips. Both debug lint reports: zero errors, 85 warnings each; warnings remain technical debt, not a clean-lint claim.
- Race verification passed: `go test -race ./internal/web ./internal/vaultwriter` (web 33.591s, vaultwriter cached).
- Store packaging/completion, immutable Store archive and combined artifact readiness gates remain pending. Live deployment remains unchanged.

## Batch

- Android wiki: folder/move/trash refresh evidence; metadata delimiter/task consistency and empty properties; contextual note/image resolution and reduced note-lookup allocation.
- Web wiki: contextual indexed note links, heading and standard Markdown note navigation, safer image selection, collapsed raw YAML properties, pipe tables and revision-checked task controls.
- Verification: repeatable browser wiki rendering/task fixtures and server stale-task-versus-external-edit regression coverage.

## Pre-version-bump verification

Full Go tests/vet, module checks, all 530 web tests, and wiki rendering, task saves, playlist forms and vault-mutation browser fixtures passed at 1440px and 390px. Android debug tasks were up to date; XML reports contain 766 tests per flavor with zero failures/errors/skips. Recent changed Android suites previously executed successfully; no claim that this last invocation re-executed them.

Current-release-only version metadata/changelog is now prepared. Next: rerun metadata tests and candidate Android debug tasks, lint, optimized Direct/Store packaging, explicit version/signature/privacy gates, immutable archives, isolated authenticated update smoke and server health checks before deployment. Preserve current signing material and rollback image/archive. Never certify old output files based on existence alone.

Physical mobile/TV testing remains unavailable. Browser fixtures are focused checks, not full live-shell acceptance. File-vault history/restore, fuller Markdown grammar and other roadmap work remain incomplete; this is not goal completion.
