# Android 0.16.86 / code 247 release verification

Status: published 0.16.86/code 247. Preparation notes below are chronological.

## Published outcome

Deployed at 2026-09-08T18:13:27.815376214Z. The running image matches the tested
candidate `sha256:dfcdd82df0c89c142b1694cbb0a7ff66cbe8546323c43467c8c1d7c82a4094d2`.
Health endpoint and Docker health both pass; restart count is zero. All ten
libraries loaded, with only the previously known malformed EPUB warning.
The live Direct APK mount hash matches the verified archive below.

The complete native invocation succeeded in 13m 39s. Direct and Store APKs
both pass the compatibility and artifact readiness gates. Store archive:
`artifacts/releases/bobobooks-0.16.86-store.apk`, SHA-256
`53c2ba139b4935ae642b5b511f7754701916a1c9a9c88edf35d284acb9907873`.
Both APKs retain the existing certificate. Previous APK archives and
`opdshelf:navigation-scan-20260908` remain available for rollback.

No personal media was changed. Physical-device acceptance and store approval
are not implied; the broad application goal remains active.

## Batch

- TV article-aware alphabet labels, filtered-view jump eligibility and selected
  semantics; shared server/web/native navigation contract.
- Wiki offline image read-back checks enforce the existing individual, total
  and count limits instead of relying on an earlier cache write.
- Durable audio checkpoints for departing items and service destruction.
- Companion web batch: action menus follow nested scrolling, only open menus
  participate in repositioning, and listeners support disposal.

The mobile layout is preserved. Hardware acceptance remains outstanding.
No old release archive or signing key is to be replaced.

## Build and gates

Source version and current-batch update notes have been prepared. Pending:
fresh tests, lint, both optimized APKs, actual version/certificate verification,
artifact store-readiness gate, authenticated update/download smoke, isolated
server checks and deployment. Passing source tests is not APK verification or
Play Store approval.

Gradle log: `/tmp/bobobooks-01686-native-release.log`.
Full Go verification: `/tmp/bobobooks-01686-go.log` and
`/tmp/bobobooks-01686-vet.log`.

## Preparation checkpoint

Full Go tests, vet and the source-only store-readiness gate passed after updating
the release-note regression assertion to this batch. Both fresh Android test
tasks completed. The same Gradle invocation is still running lint, followed by
optimized packaging; execution session 82211 is the live handle. Do not launch
a duplicate build because a poll returns no output. Artifact verification and
deployment remain outstanding.

## Candidate server and fresh test results

Both fresh native suites report 861 tests, zero failures, errors or skips.
The existing Gradle session 82211 remains live, now in Store lint after Direct
lint analysis. Optimized packaging follows in the same invocation.

The isolated server build completed as
`sha256:dfcdd82df0c89c142b1694cbb0a7ff66cbe8546323c43467c8c1d7c82a4094d2`,
tagged `opdshelf:android-0.16.86-candidate`. Build log:
`/tmp/bobobooks-01686-image.log`.

Real-server wiki checks passed at 1440px and 390px using only the synthetic
Fixture notes vault. The fixture container was stopped afterward. Playlist
forms, action-menu nested scrolling and navigation module browser checks also
passed at both widths, as did the module gate. Logs:
`/tmp/bobobooks-01686-{wiki,playlist,menu,navigation}-browser.log`.

The live server/download remain unchanged. Candidate update-download checks
must use the newly verified 0.16.86 APK, not the previous archive.

## Lint and compatibility gate

Both fresh lint reports show zero errors and 85 warnings. Gradle session 82211
has advanced to optimized Direct packaging/R8; it has not been restarted.

`scripts/verify-android-update-artifact.mjs` now checks real APK signatures,
package identity, expected version/name and strictly increasing version code
against a previous archive. It is read-only and requires explicit tool paths.
The existing 0.16.84-to-0.16.85 archives pass; deliberately mismatched expected
version/name cases are rejected. This does not yet verify the pending 0.16.86
artifacts. Certificate rotation is intentionally rejected by this gate rather
than assumed safe without a separately reviewed migration.

## Direct artifact verified

The completed Direct optimized APK reports 0.16.86/code 247 and passes the
in-place update compatibility gate against the previous Direct archive.
Certificate SHA-256 remains
`5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`.
Archived without replacing older releases:
`artifacts/releases/bobobooks-0.16.86-direct.apk`, SHA-256
`67a31e56857fe309fa1d065917bbdab3867aeeb6836f1543d2b2dfe835bc394d`.
Archive and build-output hashes match.

The candidate server passed authenticated update metadata, full-download hash,
HEAD and suffix-range checks against this actual archive. Conditional playlist
mutation checks also passed. Logs: `/tmp/bobobooks-01686-update-smoke.log` and
`/tmp/bobobooks-01686-playlist-api.log`. The fixture was stopped afterward.

Store optimization remains in the same live build session 82211. Both-artifact
readiness and publication remain pending; no live download mount was changed.
