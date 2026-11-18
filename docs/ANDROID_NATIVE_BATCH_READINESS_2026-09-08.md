# Pending native reliability batch

## Published outcome — 0.16.84 / code 245

Published September 8, 2026 at 16:03 UTC. This outcome supersedes the chronological
preparation notes below and pending-release status in this batch's implementation reports.

- Both optimized assemblies passed in 9m10s; both versions/codes and the established
  signing certificate were verified, then the combined distribution gate passed.
- Direct SHA-256: `60c3e0bbe6f076792e6af42d54cdba160e50291c83215671ef8a5cdf680c3c9a`.
- Store SHA-256: `7e063a8e8cae738cb6fa0f292171cc26f5c245381603e766aa2a1e1f99e38c1e`.
- Live image: `sha256:379b103dfeaae18601eb8d26b5fa56b2a7157e6362902041dd871a5168b2f189`.
- Started `2026-09-08T16:03:35.818566304Z`, healthy with zero restarts and all ten
  libraries ready. Live mounted APK hash matches the verified Direct archive.
- Candidate authenticated full download, HEAD and suffix-range verification passed.
- 842 Android tests per variant, both lint variants, Go tests/vet, 563 frontend
  tests, module gate and disposable wiki/playlist browser/server checks passed.
- Previous APKs and images retained. Only Mochil redeployed; personal media
  unchanged. The known malformed EPUB remains the sole scan warning.

This release does not close the broader roadmap or physical-device acceptance.

## Preparation chronology

Current deployed APK remains 0.16.83 / code 244. No version or download metadata
has been changed for this batch. Previous releases and signing material remain intact.

## Included source changes

- Serialized favorite writes, optimistic rollback and library-switch retention.
- Music Back-state cleanup and root/detail pagination isolation.
- Cancellation of obsolete catalogue jobs without cancelling independent downloads.
- TV visible-content empty-state checks and artist-album focus dependencies.
- Sleep expiry clears playback intent even during buffering.
- Process-wide wiki image-cache serialization across overlapping instances.
- HTTP-level audio-progress conflict exchange regression tests.

Existing mobile layout is preserved. Related dated implementation reports contain
the exact scope and limits of each change.

## Verified gates

- Latest complete Direct and Store debug unit reports: 842 tests each, zero
  failures or errors. Last full run completed in 15 seconds.
- Module-size/responsibility gate passed.
- Store-readiness gate passed against current source and explicitly supplied
  archived 0.16.83 Direct/Store APKs. This confirms the previous artifacts, not
  yet-unbuilt binaries containing this batch.
- Full Go tests, Go vet, frontend tests and module gate passed again for the
  release baseline. Logs: `/tmp/bobobooks-next-release-{go,vet,web}.log`.
- Existing signing key is present. Live rollback image remains
  `sha256:b6cc6cab668c3d81766dd81f6d689a22348e6fb247b160a80c50335c8c98cc11`.

## In progress / remaining

Both flavor lint tasks completed successfully under session 12486, log
`/tmp/bobobooks-native-batch-lint.log`: zero errors and 85 warnings per flavor.
Warning categories/counts match the previous report: 74 UseKtx, three LogNotTimber,
two UsableSpace, and six dependency/version advisories. No blanket suppressions added.

Source version is now 0.16.84 / code 245 with current-release-only notes. Live
download remains 0.16.83 until fresh artifacts pass acceptance.
Combined optimized assembly is running under session 73305,
`/tmp/bobobooks-01684-build.log`. Targeted server tests then candidate image build
run under session 50345, logs `/tmp/bobobooks-01684-{web-go,image}.log` and tag
`opdshelf:android-0.16.84`. Revalidate these handles; do not duplicate builds.

Candidate server build session 50345 completed successfully. Image:
`sha256:379b103dfeaae18601eb8d26b5fa56b2a7157e6362902041dd871a5168b2f189`.
Disposable server checks passed: wiki task persistence, properties, tables and
heading navigation at 1440px/390px, and authenticated playlist revision/rename/
delete checks. Both fixture containers were stopped. No personal media mounted.
Android assembly session 73305 remains active; artifact acceptance is still pending.

Direct assembly has completed and was inspected before archiving as
`artifacts/releases/bobobooks-0.16.84-direct.apk`: version 0.16.84 / code 245,
established certificate SHA-256
`5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`.
Direct artifact SHA-256:
`60c3e0bbe6f076792e6af42d54cdba160e50291c83215671ef8a5cdf680c3c9a`.
Authenticated candidate update metadata, complete download/hash, HEAD and suffix
range checks passed with this actual new archive (7,062,376 bytes). Disposable
update fixture stopped. Store optimization continues under session 73305; combined
artifact gate and deployment are still required. Live version remains 0.16.83.

After lint: prepare a fresh version and current-release-only notes, optimized
Direct/Store assemblies, exact signature/version inspection, new artifact gate,
candidate download verification and deployment health checks. Physical-device TV
focus, buffering and rendering acceptance remain unverified and are not replaced
by these automated checks.
