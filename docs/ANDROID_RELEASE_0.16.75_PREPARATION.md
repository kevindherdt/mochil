# Android 0.16.75 candidate preparation

**Published September 7, 2026 at 18:47 UTC.** The final publication record below supersedes chronological preparation statuses.

Status: pre-packaging verification. Published version remains 0.16.74 (235). Candidate source manifests now specify 0.16.75 (236), with current-release-only notes. No APK mount, signing key, or deployed container has been changed for this candidate. Do not deploy the candidate server metadata until the matching APK is verified and archived.

## Candidate scope

- Queue edits wait for a valid initial saved queue rather than replacing an unknown queue. Failed saves retain changes, with explicit mobile retry and honest loading/empty states.
- Playlist summary number validation rejects malformed counts and versions.
- Wiki draft recovery and editor limits no longer silently truncate text. Oversized edits preserve the previous text and selection.
- Wiki page responses require matching identity, actual text, and a revision before enabling writes.
- Wiki tasks skip checkbox examples in fenced code; BOM-prefixed headings/frontmatter are understood without rewriting source.
- Paired server fixes preserve exact vault Markdown bytes for revision-checked saves and detect external text changes in conditional HTTP responses.

## Evidence collected

- Full Android Direct and Store debug unit suites: **641 tests each, zero failures/errors**.
- Full Go test suite passed after the vault round-trip changes.
- Go vet passed.
- Frontend: **461 tests passed**.
- Isolated rendered playlist/queue browser fixture passed at 1440px and 390px. This does not prove full-app visual quality.
- Module boundary and store-readiness checks passed. Existing artifact checks apply to the previously built APK, not a new candidate.
- Latest Android lint is running; its completion must be checked before recording a result.

## Remaining release work

1. Finish current lint and inspect errors/warnings.
2. Update candidate version and current-release-only notes, with distribution tests.
3. Build optimized Direct and Store variants using the existing persistent signing key.
4. Verify signatures, package/version metadata, artifact gates, and archive immutable release files.
5. Stage and smoke-test the paired server image without personal library mutations; only then publish matching APK metadata and artifact.
6. Physical Android mobile/TV and TalkBack acceptance remain unverified because no device is attached. Do not describe these as tested.

This batch does not complete the wider roadmap. Native wiki revision history, broader TV device acceptance, and further cross-client feature work remain open. No mobile redesign is included.

## Packaging preparation status

Candidate metadata and distribution-note tests passed. Direct lint has completed successfully; Store lint is still running in the same Gradle invocation. The lint invocation began before the version-only source bump; optimized packaging will verify the candidate manifest separately.

A server image build was started under `opdshelf:android-0.16.75`, deliberately not `opdshelf:local`. Build completion and isolated smoke testing remain pending. The live container still mounts the immutable 0.16.74 archive and reports zero restarts. Optimized Android packaging has not yet started.

Subsequently both lint tasks completed successfully in 4m37s. Optimized Direct and Store packaging has now started in one Gradle invocation, using the existing persistent signing configuration. Build results and signatures are not yet verified.

## Staged server smoke

The server build completed under `opdshelf:android-0.16.75`, image ID `sha256:7e8412b6edbdeac84a89b606ddfe0f4df1b6cd972d30e12299fc580c96750905`.

An isolated `--rm` container on loopback port 18102 passed health, empty playlist summary, and synthetic percent/slash playlist-name PUT/GET checks. The first synthetic PUT used the wrong object-shaped payload and correctly returned 400; retrying with the documented array payload succeeded. `/proc/1/status` confirmed the application UID is 65532 (a separate docker-exec shell defaults to root and is not evidence of application privilege). The test container had zero restarts and was stopped/automatically removed, including its disposable test state. No personal config/media mounts were used.

The staged image is not tagged live or deployed. Android optimized packaging remains active; no candidate APK has yet been verified or published.

## Additional release checks

Full Direct/Store debug lint reports each contain zero errors and 84 warnings. Race-enabled tests passed for `internal/document`, `internal/vaultwriter`, and `internal/web` (web completed in 29.402s). The live server remains healthy with zero restarts; the archived/published Direct 0.16.74 SHA-256 remains `1c90ee59a4a5d71be109edcbe11f72d995333a5a4864f313861fe5a6086f9fc7`.

The active optimized build has reached Direct vital lint after R8. Existing APK files in the output directories are not evidence of candidate completion; wait for packaging and inspect their actual version/signature before archiving.

## Direct candidate verified

Direct optimized assembly completed. APK metadata reports `net.kdhmedia.bobobooks`, version 0.16.75, code 236. Signature verification succeeded with certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`, matching prior releases. This is the existing persistent development-format certificate, not a new Play Store production-signing decision.

The explicit artifact readiness gate passed. Archived without overwriting any existing release as `artifacts/releases/bobobooks-0.16.75-direct.apk`; archive and build output compare identically. SHA-256: `5b57941c8dd45567701422944a66488e51c8b2284c055abc33540deff982d5da`.

Store optimized R8 is still running in the original Gradle job. Direct is not yet mounted/published; physical in-place installation remains unverified without a device.

## Both candidates verified

The combined optimized build completed successfully in 8m31s. Store version 0.16.75/code236 passes signature verification with the same certificate and the explicit artifact gate. Archived as `artifacts/releases/bobobooks-0.16.75-store.apk`, SHA-256 `d603983edb57d970f9ac0792b2c8f3a1af3a5c895fccb6bc5e79e086619cc3eb`.

The staged server's browser download endpoint returned version 0.16.75 and exactly the archived Direct SHA-256. An isolated test-device login required configuring the test public URL; Android update metadata was then checked with the returned `device_token`. Initial requests without authentication correctly returned 401. No real user credentials or personal config were used. Disposable smoke containers were stopped for automatic removal.

Both APKs remain unpublished. The matching staged server image and Direct archive are ready for coordinated publication, followed by live health/version/hash checks.

## Final publication

Published the verified Direct archive and matching server image together using `docker compose up -d --no-build --no-deps opdshelf`. The container started at `2026-09-07T18:47:55.512026598Z`, reports healthy with zero restarts, and `/healthz` returns `ok`.

- Live image: `sha256:7e8412b6edbdeac84a89b606ddfe0f4df1b6cd972d30e12299fc580c96750905`.
- Mounted APK: `/downloads/bobobooks-android.apk`, Direct 0.16.75/code236, SHA-256 `5b57941c8dd45567701422944a66488e51c8b2284c055abc33540deff982d5da` (verified inside the running container).
- All ten libraries loaded. The existing malformed EPUB warning for a missing `META-INF/container.xml` remains; no new startup failures appeared.
- Older archived releases and the previous tagged image remain available. No personal media/config content was edited. Only the APK bind-mount target changed.
- Store APK remains a private verification artifact, not a Play Store publication. No attached-device installation or TV remote/TalkBack acceptance was claimed.
