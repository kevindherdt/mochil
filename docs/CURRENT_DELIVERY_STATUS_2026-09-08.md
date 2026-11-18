# Current delivery status

This checkpoint separates inspected current state from historical roadmap notes.
It does not declare the broad application goal complete.

## Published

- Android 0.16.86/code 247 and companion server/web release, deployed at
  18:13 UTC: TV bucket navigation/semantics, wiki cache read limits, durable
  departing-item/shutdown audio checkpoints and nested-scroll action menus.
  See `ANDROID_0_16_86_RELEASE_VERIFICATION.md` for artifact and live evidence.
- Android 0.16.85/code 246: wiki tree ordering/memory improvements, revision-based
  image reuse, page-close loading cleanup and music jump state corrections.
- Server grouping-performance image, started 2026-09-08 at 17:19 UTC: cached
  normalization keys, shared disc-folder rules, reduced grouping allocations,
  publication normalization reuse and provider-ID punctuation protection.
- Server/web navigation-scan batch, started 2026-09-08 at 17:43 UTC: shared
  web alphabet-bucket rules, ignored-sidecar scan savings and uppercase audio
  fallback-title correction. Includes the preceding grouping-performance fixes.
  Live image and verification: `NAVIGATION_SCAN_DEPLOYMENT_2026-09-08.md`.
- Executable extension integrations remain removed. The source/artifact gate
  passed for the existing release; this is not Play Store approval.

Release evidence lives in `ANDROID_PENDING_BATCH_VERIFICATION_2026-09-08.md`
and `MUSIC_GROUP_PROJECTION_COST_2026-09-08.md`. Their top-level published
outcomes supersede the chronological pending notes beneath them.

## Latest batch status

Published server follow-up at 18:31 UTC: prepared title keys for audio sorting, measured at
60–64% less sort time for 5,000 synthetic tracks. Equivalence tests, full Go
tests, vet and module gate passed. See `AUDIO_TITLE_SORT_COST_2026-09-08.md`.

Also published: unfiltered audio pages skip unused full-account progress snapshots
while retaining spoken-media progress and played filters. See
`AUDIO_HISTORY_SNAPSHOT_COST_2026-09-08.md` for the scoped benchmark and tests.
Deployment evidence: `AUDIO_COST_DEPLOYMENT_2026-09-08.md`.

Published server follow-up at 18:22 UTC: podcast duration parsing now lives in the podcast
domain and rejects non-finite/negative/overflowing input before JSON projection.
Go tests, vet and module gate passed. See
`PODCAST_DURATION_BOUNDARY_2026-09-08.md`.

Also published: deterministic audio title ties and correct
volume ordering for equivalent series spellings, with sorting policy extracted
from the HTTP handler. See `AUDIO_NORMALIZED_SORT_TIES_2026-09-08.md`.
Deployment evidence: `AUDIO_VALIDATION_DEPLOYMENT_2026-09-08.md`.

The previously pending TV, wiki cache, audio lifecycle and web action-menu
changes are published in 0.16.86. No unpublished implementation from that batch
remains. Hardware acceptance remains separate from delivery.

The concurrent durable audio progress regression is test-only (20 race-detector
repetitions), not a pending runtime feature. The shared bucket corpus is verified
across Go/web/TV; its native TV behaviour is included in the published APK.

No need to rebuild Android merely to publish the server-only scanner changes.
No new APK is warranted solely by the shared verification files.

## Confirmed feature gaps

### File-backed wiki history and restore

`internal/web/vault_http.go` and Android `WikiClient.kt` expose current vault
CRUD, not revision history. Existing legacy `/api/wiki` history must not be
misrepresented as vault history. Before implementation, choose opt-in/default,
retention limits, private storage ownership and external-edit conflict behavior.
Do not delete the old managed store without a migration path.

### Broader wiki offline attachments

`WikiOfflineMirror.kt` mirrors bounded Markdown and referenced images. PDF
opening uses a separate bounded cache; that is not full-vault attachment
mirroring. Additional formats need bounded storage, MIME verification and safe
offline-opening behavior. Keep availability claims limited to saved content.

### Physical acceptance

No connected device was available at the TV audit. Real remote focus, overscan,
font scaling, playback interruptions, cross-device reading/highlight behaviour
and e-ink presentation cannot be signed off from JVM or browser tests.
Preserve the user-approved mobile layout during this work.

## Next safe work

1. Navigation browser verification and the bounded server/web deployment are
   complete; keep the existing APK intact until the next meaningful native batch.
2. Continue focused web music interaction checks rather than redesigning the
   shell or filling empty space with counters/widgets.
3. Audit wiki attachment ownership, bounds and offline failure paths before
   expanding supported offline media.
4. Batch further Android fixes before another optimized APK build; leave
   hardware acceptance and history-retention choices for user feedback.

This is a scoped checkpoint, not an exhaustive feature-completion audit. The
remaining roadmap still governs broader desktop, TV, metadata/ingest and future
client work; unchecked historical phrasing alone is not proof of missing code.
