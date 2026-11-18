# Post-0.16.79 regression batch

Status: this batch was published as 0.16.80/code241 after completed optimized builds, artifact/signature gates and isolated server/update tests. See `ANDROID_RELEASE_0.16.80_PREPARATION.md` for hashes, deployment checks and remaining limitations.

## Included work

- Native wiki PDF link opening: bounded authenticated cache, cancellation and stale-opening guards, native reader reuse, local-only progress, relative Markdown links and small existing-layout loading feedback.
- Wiki exact-case attachment matching, missing-note path validation and lazy search-order caching. Host-JVM benchmark methodology/results are in `WIKI_SEARCH_ORDER_CACHE_2026-09-08.md`.
- Web progress queue recovery after transport errors and old-item response isolation for audio/video.
- Sequential web playlist edit intent and identity-based drag ordering; focused drag bindings extracted from the audio-library coordinator.
- Shared Android encryption-key initialization lock; existing keys/formats/aliases preserved.
- Native HTTP progress recovery and authenticated server PDF attachment regression coverage.

## Completed checks

- `go test ./...` and `go vet ./...` exited zero.
- Full frontend suite: 540 passed, zero failed.
- Module boundaries/budgets and distribution-readiness script exited zero.
- Playlist forms at 1440/390px, desktop native HTML drag gestures across delayed saves, wiki properties/table/heading fixtures and task save/recovery at 1440/390px passed.
- ADB reports no connected devices. No physical mobile/TV acceptance is claimed.

## Android process

Full Direct/Store debug unit tests and lint completed successfully in 4m36s. Each flavor executed 786 tests with zero failures/errors/skips. Log: `/tmp/bobobooks-next-batch-android.log`. Go/web session 51396 and browser session 33430 both completed with exit zero.

Do not certify release artifacts from existing APK files: those are still the prior build. Next release needs version/changelog preparation, current optimized builds, explicit version/signature/archive checks and isolated server/APK update tests before publishing. Preserve the established signing identity and rollback archives. The broader roadmap and physical-device checks remain incomplete.
