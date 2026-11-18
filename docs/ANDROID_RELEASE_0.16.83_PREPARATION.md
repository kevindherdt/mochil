# Android 0.16.83 / code 244

Status: published September 8, 2026 at 15:02 UTC. This outcome supersedes the
chronological preparation notes below.

## Published outcome

- Image `sha256:487626ef620cb7b8e95df5a88ab61867bd7f933707dfaeeda756c1854e40a714`,
  started `2026-09-08T15:02:44.069998814Z`; healthy, zero restarts, ten libraries ready.
- Both optimized assemblies passed in 8m50s. Version 0.16.83/code244 and established
  signing certificate verified for both archives; combined distribution gate passed.
- Direct SHA-256 `a7e1d9af68055cfa0c78cfdb51db77ff95587114e5ad59b0a84181d8924d7bee`.
- Store SHA-256 `a469cca1b50f83fd6cca3f5c77ed23a19e4a60d83f79bd2a57c79c74100112e1`.
- Candidate authenticated download, HEAD and range checks passed. The live mounted
  APK hash matches the Direct archive. No new live device session was created.
- Go tests/vet, 561 frontend tests, 815 Android unit tests per flavor, lint,
  module gate and browser fixture checks passed.
- Previous APKs and images retained. Only Mochil redeployed; no personal media
  changed. The known malformed EPUB remains the sole library scan warning.
- Physical-device acceptance and broad roadmap work remain outstanding.

## Scope

- Identity-scoped watched updates, stale mutation-error suppression, immediate
  search invalidation, coalesced search and pagination cancellation for videos.
- Compatible lower-cost wiki image-cache lookup and persisted-snapshot asset
  planning that includes retained notes during partial refreshes.
- PDF cache eviction only after successful replacement; failed downloads preserve
  usable offline attachments.

No mobile layout changes. Current-release-only notes are in the server metadata.

## Required gates

Both unit suites, Android lint and optimized assemblies; Go tests/vet, frontend
tests and module-size checks; exact APK version/code/signing identity; explicit
Direct/Store distribution checks; candidate authenticated update/download hash
and browser regression fixtures; live health and download verification after
deployment. Keep existing signing key, archives and rollback image unchanged.

Physical-device acceptance and the broad app goal remain incomplete. Pulse and
all personal media are out of scope for this release.

## Running verification

Android pipeline session 9333 runs both flavor unit suites and lint, followed by
both optimized assemblies only on success. Logs: `/tmp/bobobooks-01683-checks.log`
and `/tmp/bobobooks-01683-build.log`. Continue this handle; do not start a duplicate.

Full Go tests and vet passed. The first frontend command used nonexistent `npm
test` and stopped; it was corrected to the actual `npm run test:frontend` script.
The corrected frontend/module pipeline builds candidate tag
`opdshelf:android-0.16.83` only if those checks pass.

Corrected pipeline session 53004: all 560 frontend tests and the module-size gate
passed; candidate Docker build is running. Revalidate this handle rather than
starting another image build. Android session 9333 remains in lint analysis.

Live image remains `sha256:e33c495a81eab6af8853db8287b25ff5c90e939230d854d0b59809977619433b`,
healthy with zero restarts. No new artifacts are archived or published yet.

## Candidate server verified

Image build session 53004 completed successfully. Candidate image:
`sha256:487626ef620cb7b8e95df5a88ab61867bd7f933707dfaeeda756c1854e40a714`.

Disposable fixture checks (session 44098) passed: wiki task persistence, properties,
tables and heading navigation at 1440px and 390px; authenticated playlist revision,
rename/delete and duplicate-create behavior; packaged audiobook chapter controls
at both widths. Fixture containers were stopped afterward; no personal data mounted.

Fresh Android reports contain 815 tests per flavor with zero failures/errors.
Session 9333 remains active in lint. APK assembly/version/signature/archive and
authenticated update byte/hash checks remain required before publishing.

## Lint passed; optimized assembly running

Session 9333 passed unit/lint verification in 5m23s and has advanced to combined
optimized assembly (`/tmp/bobobooks-01683-build.log`). Keep following this session.

The update smoke script now refuses the live 8099 port and tests authenticated
HEAD and suffix byte-range delivery in addition to full-download SHA-256. Its
new safety regression passed. A disposable candidate transport check passed using
the **old 0.16.82 APK solely as fixture bytes**. That is not 0.16.83 APK acceptance:
the actual fresh artifact still needs package/signature inspection and this same
download check. The transport fixture container was stopped afterward.

## Additional acceptance and rollback checks

All 561 frontend tests passed after the update-smoke safety regression was added.
Browser playlist drag (successive gestures during a delayed save) and video
progress retry/close/pause at 1440px and 390px also passed. These are browser fixture
checks, not native-device acceptance.

The two 0.16.82 archive hashes and both retained rollback image IDs were rechecked
unchanged. Session 9333 remains active; Direct R8 completed and packaging/vital lint
is underway. Existing output APKs were still 0.16.82 when inspected, so neither
has been archived under the new version. No live settings were changed.

## Direct artifact accepted

Fresh Direct APK verifies 0.16.83/code244 and certificate SHA-256
`5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`.
Archived without replacing any existing release at
`artifacts/releases/bobobooks-0.16.83-direct.apk` (7,062,376 bytes), SHA-256
`a7e1d9af68055cfa0c78cfdb51db77ff95587114e5ad59b0a84181d8924d7bee`.

The actual archived new APK passed candidate authenticated metadata, full-download
hash, HEAD and suffix range checks. Its disposable container was stopped. Store
assembly remains active in session 9333; combined artifact gate and live deployment
remain pending.

## Store assembly wait

The Direct-only distribution check and module-size gate passed. Store R8 is still
running in session 9333; the Store output last inspected is still 0.16.82/code243
and must not be archived as the candidate. Live health and zero restarts were
rechecked while waiting. No additional build was launched.
