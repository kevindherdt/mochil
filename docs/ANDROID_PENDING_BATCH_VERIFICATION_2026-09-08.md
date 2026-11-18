# Android pending batch verification

Status: published 0.16.85 (246). Earlier pending notes below are chronology.

## Published outcome

Deployed the verified candidate at 2026-09-08T17:03:40.816505842Z.
`opds-opdshelf-1` reports healthy, zero restarts, and image
`sha256:eaf783b05cdf035c74bca9200869e9ed8c9c65c861425bcf97d7d4fd5fa0a536`.
The live APK mount hash matches the verified Direct archive below. All ten
libraries loaded; the only scan warning is the previously known malformed
The Regressor and the Blind Saint EPUB (missing META-INF/container.xml).

Rollback image `opdshelf:artwork-20260908` and both 0.16.84 APK archives remain
available. No personal media was modified, and unrelated containers were not
restarted. Physical-device acceptance remains outstanding; this release is not
a claim that the full application goal or store approval is complete.

## Candidate verified; deployment pending

Both optimized builds finished successfully in 9m 15s. Both APKs report
0.16.85/code 246 and verify against the existing signing certificate recorded
below. The packaged-artifact store-readiness and module gates passed.

- Direct archive SHA-256:
  `6c7e240bf3423c1bc9f12bc96377b2bf35ff8c131ae710aaac20399d886d2e7b`
- Store archive SHA-256:
  `fd5289b41a44f2a5c93066a3bd3b3311f73a927c0e755f6fc594c2d530b334b7`
- Candidate image:
  `sha256:eaf783b05cdf035c74bca9200869e9ed8c9c65c861425bcf97d7d4fd5fa0a536`

Authenticated update smoke passed against the real Direct archive (7,062,376
bytes): version/changelog response, full hash, HEAD and final-256-byte range.
Log: `/tmp/bobobooks-01685-update-smoke.log`. The disposable update fixture was
stopped. Final full Go tests and vet passed; logs:
`/tmp/bobobooks-01685-final-{go,vet}.log`.

The chronology below records earlier in-progress states. Publishing the candidate
and changing the live APK mount remain outstanding; physical device acceptance
and Play Store approval are not implied by these gates.

## Release candidate in progress

Source version is now 0.16.85 (246), with update notes restricted to this batch.
The update-info/download handler tests pass. Both optimized variants were
started using the existing persistent signing configuration; build output is
`/tmp/bobobooks-0.16.85-optimized.log`. Do not archive, serve or label an APK as
0.16.85 until its actual version and certificate have been verified.

The matching isolated server image build is tagged
`opdshelf:android-0.16.85-candidate`, with log
`/tmp/bobobooks-0.16.85-image.log`. Neither build implies deployment. Live APK
mounts and the running container remain unchanged until candidate verification.

The server candidate build completed successfully. Its real-server wiki smoke
passed at 1440px and 390px, covering persisted task toggles, properties, tables,
heading navigation, horizontal overflow and uncaught page errors. It used only
the disposable Fixture notes vault at loopback port 18102; no production media
was mounted. Log: `/tmp/bobobooks-01685-wiki-browser.log`. The temporary fixture
container was stopped after the test. Optimized APK verification remains pending.

The authenticated candidate playlist API checks also passed: duplicate create,
conditional replacement, case-only rename and stale-revision delete rejection.
Log: `/tmp/bobobooks-01685-playlist-api.log`. That synthetic fixture was stopped.

The Direct optimized artifact now identifies itself as 0.16.85/code 246 and its
signature verifies with the existing certificate SHA-256
`5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`.
It is archived as `artifacts/releases/bobobooks-0.16.85-direct.apk` without
overwriting prior release archives. The Store optimized build is still running;
actual authenticated APK download checks and deployment remain outstanding.

## Included changes since the installed release

- Wiki image mirroring reuses unchanged revision-addressed images, retaining
  owner/vault isolation and the existing image-count and byte limits.
- Wiki natural ordering precomputes name segments and uses iterative traversal
  rather than recursive calls for deeply nested folders.
- Wiki tree visibility retains integer parent links instead of ancestor lists.
- Closing a Wiki page clears obsolete loading state and cancels its tree job.
- Music letter jumps reject offsets belonging to a different content projection
  and clear obsolete pagination state when replacing the current result.

The server's grouped music navigation, deterministic edition ordering, terminal
pagination and complete-compilation search corrections are also pending. See
`MUSIC_GROUP_NAVIGATION_ALIGNMENT_2026-09-08.md`; publish these with or before the
native letter-jump fix.

## Fresh verification

Ran both Gradle unit test tasks after their corresponding clean-test tasks, so
the test execution was not accepted merely from an up-to-date cache result:

```
./gradlew :app:cleanTestDirectDebugUnitTest :app:cleanTestStoreDebugUnitTest \
  :app:testDirectDebugUnitTest :app:testStoreDebugUnitTest --console=plain
```

Build successful in 12 seconds. XML reports record 857 tests for Direct and 857
for Store, each with zero failures, errors and skipped tests. Compilation inputs
were up to date; both test tasks executed. Log:
`/tmp/bobobooks-native-batch-fresh-tests.log`.

## Still required before publishing this batch

- Optimized build verification (both-variant debug lint passed below).
- A new version/code, current-batch changelog and original signing certificate.
- Verify the actual APK archives and the server's update/download response.
- Isolated candidate-server checks before deployment.

Physical Android TV focus/navigation and device rendering are not proved by
these JVM tests. No TV acceptance claim, new APK or deployment is implied by
this report. The established mobile layout has not been redesigned.

## Cross-platform follow-up gates

- Current frontend unit suite: 563 passed, no failures or skips.
- Synthetic playlist browser flow: passed at 1440px and 390px. This is a
  functional form/queue check, not a complete visual audit of the music library.
- Store-readiness script: passed against current source and the explicitly
  supplied existing 0.16.84 Direct/Store archives. It does not verify an as-yet
  unbuilt new APK and does not constitute store approval.

Logs: `/tmp/bobobooks-pending-web-tests.log`,
`/tmp/bobobooks-pending-playlist-browser.log`,
`/tmp/bobobooks-pending-store-readiness.log`.

Both-variant debug lint completed successfully in 5m 6s, with zero errors and
85 warnings per variant. Log: `/tmp/bobobooks-pending-batch-lint.log`.
The Direct warning categories are 74 UseKtx convenience suggestions, three
LogNotTimber notices, two UsableSpace suggestions and six build/dependency-version
notices. These are retained findings, not suppressed or represented as zero
warnings. Broad dependency upgrades and changing the conservative free-space
policy are not bundled into this stabilization release without their own tests.
