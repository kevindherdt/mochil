# Mochil modularity audit

Status: active engineering constraint. Updated 2026-08-28.

September 7 follow-up: audio queue, playlist, and progress stores now share the
atomic JSON writer in `internal/audio/state_io.go`. Locking, validation, mutation,
and rollback remain in their respective stores. The writer enforces each
decoder's existing byte budget (4 MiB collections, 64 MiB progress) before
replacing the live file; previously these writers could save unreadable oversized
state. Existing temporary filename patterns, private permissions, file fsync,
and rename behavior are preserved. This removes three copies of filesystem
transaction code without creating a generic persistence framework. Tests cover
oversize rejection, prior-file preservation, temporary cleanup, private modes,
collection memory rollback, and progress's distinct larger budget. Encoding
still uses Go's JSON encoder; the file-size bound is not a peak-memory bound.
Deployed September 7 at 16:13 UTC; see `STATE_WRITE_BOUNDS_2026-09-07.md`.

Android Wiki's offline asset budget now has a pure owner in
`WikiOfflineAssetBudget.kt`. The image store budgets the resulting files using
both retained sizes and incoming replacement sizes before pruning, rather than
assuming the current network batch's 24 MiB limit also bounds files retained
from previous mirrors. The existing 32-image, 3 MiB-per-image, and 24 MiB-total
limits remain unchanged. Tests cover mixed retained/new-sized candidates,
invalid sizes, duplicate identities, count limits, and fitting smaller images
into the remaining space. Direct/Store Wiki suites and the module gate pass.
No current device cache was modified; this is pending the next APK batch.

The same Wiki pass moves bounded image-batch orchestration into the existing
`WikiOfflineAssetLoad.kt` owner. Once retained download bytes reach 24 MiB,
later batches are not requested; missing images still allow subsequent batches,
and cancellation propagates. A synthetic 32-image case with 3 MiB images now
requests eight images instead of the previous loop's 32, retaining the same
eight images. This is an executable request-count result, not a measured network
or battery improvement. At most four images are requested concurrently and
the 32-target ceiling remains. Direct/Store Wiki suites pass; pending APK batch.

Android mobile Settings now owns its complete destination-only state boundary in
`MobileDeviceSettingsRoute.kt`: update delivery, download location, crash and
performance diagnostics, annotation privacy/export, ReplayGain, and both system
document launchers are collected only while Settings is composed. The connected
application root no longer invalidates media browsing when administrative state
changes, and a source-level regression keeps those subscriptions out of the
composition root.

Android mobile Home now owns its dashboard state, media artwork routing, resume
offset lookup, refresh and dismissal actions in `MobileDashboardRoute.kt`.
Dashboard activity can no longer invalidate Library or Settings through the
connected application root, and a source-level regression freezes that
destination boundary. Its shared dismissal wrapper also owns the transient
swipe affordance for reading, listening, and watching rows; individual media
rows retain only their content and long-press menu behavior.

Android mobile Library now owns catalogue, music, podcast, video, and collection
state plus their media actions in `MobileLibraryRoute.kt`. Home and Settings own
their separate library-directory subscriptions, while the connected root observes
only a distinct `offlineOnly` navigation projection and the queue/playlist state
required by the persistent player. Source-level regressions prevent high-churn
media state from returning to the app root.

Android TV now follows the same destination ownership model. `TvConnectedApp.kt`
is a session and route coordinator only; `TvHomeRoute.kt`, `TvLibraryRoute.kt`,
and `TvSettingsRoute.kt` independently own dashboard, media catalogue, and
administrative state. This keeps remote-focus navigation from being invalidated
by unrelated destinations and gives each TV surface a bounded place to evolve.
The persistent TV dock remains its own presentation owner and now applies
progressive disclosure: artwork, identity, timeline, play/pause, Now Playing,
and close stay available without duplicating previous/next from the full player.
Home and Library share the same shoulder-navigation pass-through behavior when
no adjacent visible shelf exists.

Android publication rendering now separates navigation from reusable media
presentation. `PublicationScreen.kt` owns browse state, destination controls,
search, and two-pane routing; `PublicationItems.kt` owns book details, cover and
list items, authenticated artwork, progress chrome, paging sentinels, and shared
empty/error states. The coordinator falls from 436 to 261 lines because of that
responsibility boundary, and item presentation can evolve without expanding
navigation-state ownership again.

Web music destination changes now use the focused `loading-height-guard.js`
primitive when a network-backed Tracks view replaces the current catalogue.
The guard preserves only the rendered canvas height during the request and
releases it on success or failure, preventing viewport clamping without moving
loading state or destination ownership into the audio composition root.

Android publication transport now lives in `PublicationCatalogueRepository`.
It owns dispatcher switching and catalogue/collection queries, including a
single collection snapshot operation, while `LibraryViewModel` retains stale
request generations, navigation, and presentation state. The view model drops
to 420 lines and no longer imports IO dispatchers or constructs transport
clients; the architecture gate freezes that responsibility boundary.

The rendered browser smoke gate now verifies the persistent content shell as
behavior, not merely CSS intent. At desktop, phone, and 320 px compact widths
in both themes, catalogue content must move while the masthead and active
library controls retain their viewport position. The same run continues to
enforce zero horizontal overflow, 44 px touch targets, reachable compact
navigation, and audio persistence across route changes.
- Dashboard removal chrome is now governed by the dashboard stylesheet rather
  than per-media exceptions. Fine pointers reveal all three dismissal controls
  on card hover or focus-within; touch retains visible controls, keyboard focus
  remains explicit, and reduced motion disables the opacity transition. A
  focused stylesheet contract and the 346-test frontend suite protect it.

Executable extension and Suwayomi modules and surfaces are absent from the
Android and web applications. The strict configuration decoder accepts the old
field only to keep upgrades bootable, then immediately discards its payload.
Generated Android build histories and Kotlin workspace caches are excluded from
repository and container-build contexts as well; the store-readiness gate now
freezes that hygiene boundary alongside source, dependency, manifest, and APK
inspection. The gate resolves the repository from its own location so release
and CI checks do not depend on the caller's working directory.

Android destination visibility is isolated in `DestinationVisibilityPreferences.kt`; Saved and Sources do not add preference responsibilities to catalogue state. The composition root only wires the two values into settings, the virtualized rail, and a small route guard.

Android publication navigation persistence now lives in
`LibraryNavigationStore.kt`. It owns the selected shelf, grouping and layout
preferences, plus the bounded last-known library directory used for offline
startup. JSON encoding/decoding is independently tested and fails closed for
malformed or identity-less cache entries. `LibraryViewModel.kt` retains
catalogue requests, paging generations, selection transitions, collections,
and publication opening and shrinks from 508 to 486 lines; the extraction is a
responsibility boundary rather than a line-target split.

Web reader keyboard and button navigation now lives in `reader-navigation.js`. Logical Previous/Next behavior is tested independently from physical left/right shortcuts, preventing fixed-layout LTR and RTL policy from growing back into the reader coordinator.

Reading-state projections now live in `internal/reading/query.go`: account export, immutable history/get lookups, KoSync-document lookup, existence checks, deterministic ordering, and bounded newest-position selection form one read boundary. `store.go` retains persistence, revisioned mutations, history transitions, KoSync recording, and deletion and shrinks from 529 to 410 lines. Query tests and the 100,000-position benchmark live beside the extracted owner.

Wiki projections now live in `internal/wiki/query.go`: page/folder lists, individual lookup, backlinks, history, and account export form one immutable read boundary. `store.go` retains validation, persistence, revisioned page/folder mutations, trash retention, version restore, and account deletion and shrinks from 511 to 420 lines. The extraction also closes a snapshot leak by deep-copying deletion timestamps before pages cross the store lock; focused race coverage protects that contract.

## Rules

- New production modules should normally stay below 500 lines. The 800-line check is an emergency growth ceiling, not an architectural target or completion criterion.
- A touched oversized module must stay the same size or shrink; feature work may not make a grandfathered monolith larger.
- Split by product responsibility, not arbitrary line ranges. Each extracted module owns its state, rendering, events, and focused tests.
- A module is complete when it has cohesive ownership and a narrow interface—not when it falls one line below a threshold. Coordinators should remain thin even when technically below the ceiling.
- Do not create callback-heavy fragments merely to move lines. If an extraction exposes too many state callbacks, introduce a bounded state owner or facade in the next pass.
- Shared CSS contains tokens and truly shared primitives only. Feature styling belongs beside the feature (`vault.css`, reader CSS, music discovery CSS, and similar modules).
- `index.html` is a shell, not a component warehouse. Reusable and feature-specific templates should move into bounded HTML or JavaScript render modules.
- Minification is a delivery optimization, not proof of modularity. Both source ownership and delivered byte size matter.

## Current high-priority findings

No production module currently exceeds its justified composition-root boundary or the ordinary source/byte ceilings. Continue applying the same responsibility-first review whenever a module is touched.

Android canonical API redirect validation now lives in the focused network
module `AndroidApiRedirectPolicy.kt`. Wiki, publication, dashboard, audio,
podcast, and video catalogue transports share that security boundary instead
of maintaining subtly different host, scheme, port, and API-path rules.
Read-only requests may use the policy while state mutations retain separate
non-redirecting connection paths; focused tests freeze both behaviors.

Settings navigation now has a focused `settings-navigation.css` owner for its
workspace-width and compact snapping behavior. The rule overrides the legacy
shared form-width constraint without adding more Settings behavior to
`app.css`; a stylesheet-ownership contract protects the boundary.

Android interrupted-transfer policy now lives in the 101-line
`ResumableHttpDownload` owner. Validator persistence, Range/If-Range request
configuration, exact Content-Range validation, bounded append/restart behavior,
and partial cleanup are shared by publication and audio storage instead of
being duplicated in their UI-facing coordinators. The architecture gate keeps
both call sites on that shared boundary.

Metadata inspection now follows that rule explicitly: `metadata-review.js` owns
publication inspection decisions and apply confirmation, while
`music-quality-review.js` owns read-only music health reports,
`ingest-preview.js` owns cursor paging, automatic continuation, reset, and
failure recovery for structured-path proposals, while
`publication-matcher.js` owns opt-in provider lookup, inspect-only ambiguous
matches, and reversible provider-cover state. This keeps external matching and
network error handling out of the metadata job controller. The coordinator
shrunk from 547 to 412 lines, and the module guard prevents these
responsibilities from being folded back into it.

The server-side metadata boundary mirrors that separation. `metadata_http.go`
owns inspection jobs, decisions, previews, and transactional filesystem or
Calibre apply behavior. `metadata_provider_http.go` owns optional online
provider search, proxied suggestion artwork, and reversible per-publication
cover overrides. This keeps slow external requests and provider validation out
of the source-write transaction coordinator; the architecture gate prevents
those route families from being merged back together.

Large test files are lower risk because they do not create runtime coupling, but should still be split along the production boundary they verify when edited substantially.

## Work already started

- Podcast feed-format handling now lives in `internal/podcast/rss.go`: RSS XML
  decoding, stable episode identity, description sanitization, publication-date
  credibility, and enclosure normalization form one untrusted-input boundary.
  `store.go` retains subscription state, snapshots, refresh orchestration, and
  atomic persistence and falls from 488 to 330 lines. The architecture gate
  prevents parser responsibilities from returning to storage.
- Obsidian-vault hierarchy lives in `vault-tree.js`.
- Vault workspace styling lives in `vault.css`, outside the shared stylesheet.
- Code-block clipboard behavior lives in `code-copy.js` with a focused browser test.
- Theme persistence, device-color following, and appearance control bindings now live in `theme-controller.js` with focused tests; `app.js` shrank by 48 lines.
- Search lifecycle, popup dismissal, and sticky-shell measurement now live in three focused controllers with focused tests; `app.js` shrank by another 74 lines and is now below 1,000 lines.
- Dashboard, Library, Wiki, Sources, and Settings routing now lives in `app-view-controller.js`; its active-surface and navigation behavior has focused tests, and `app.js` shrank by another 42 lines.
- Catalogue-toolbar interaction now lives in `catalogue-toolbar.js`, while library discovery, saved selection, and visibility refresh live in `library-loader.js`; both boundaries have state-transition tests. This removes another 84 lines from `app.js`, but the entry point remains an active P0 until it is a thin composition root.
- Library identity, visibility, active selection, and persistence now have one owner in `library-state.js`. The loader and toolbar consume that facade instead of exchanging library state through ten getter/setter callbacks; reloads also revalidate selections after a library is removed or renamed.
- Ordinary-book catalogue state now lives in `book-catalogue.js`: bidirectional paging, bounded DOM-window bookkeeping, rendering, request cancellation, stale-response rejection, navigation loading, alphabet jumps, and scroll-position feedback share one tested owner. This stage removed 175 lines from `app.js` and established the boundary used by the subsequent series extraction.
- Series catalogue state now lives in `series-catalogue.js`: list/cover rendering, expandable and focused series, nested pagination, request cancellation, stale-response rejection, and cover cleanup share one tested owner. The extraction also fixes collection-save buttons accidentally receiving open-series behavior. At 404 lines, `app.js` is now a bounded composition root and leaves the P0 list; its frozen budget remains 404 so later feature work cannot gradually turn it back into a monolith.
- The reusable library editor now lives in `library-form-template.js` and is mounted by its owning settings controller. Contract tests cover every field consumed by settings code and idempotent mounting. This removes 158 lines from `index.html` without making the composition root aware of settings markup.
- The video player dialog now lives in `video-player-template.js` and is mounted by the video controller before binding. Its DOM/accessibility contract is tested. Track language normalization, preferred-track matching, labels, and runtime formatting moved to tested `video-format.js`, so `video-library.js` shrank from 932 to 895 lines while gaining template ownership rather than growing.
- The complete audio-player surface now lives in `audio-player-template.js` and is mounted by the audio controller before it binds playback behavior. Contract tests freeze every transport, chapter, timing, favorite, and playback-option hook; this removes another 137 lines from `index.html` without moving feature knowledge into `app.js`.
- Account-management behavior and markup now share one feature boundary: `accounts-settings.js` mounts `accounts-settings-template.js` with an explicit document dependency. Contract tests freeze the administrator-only creation, recovery, role, password, status, and registration-copy hooks. This removes another 77 lines from `index.html`; account behavior remains isolated from the global settings save lifecycle.
- Collection selection/creation and destructive metadata confirmation now each own their dialog template instead of sharing an unrelated global dialog block. Their controllers mount the dialogs before binding, with tests covering idempotence, required DOM hooks, accessibility labels, disabled confirmation, and rollback copy. This removes another 54 lines from `index.html` without introducing a generic dialog monolith.
- Library-scan activity now owns `activity-panel-template.js`, while the home dashboard owns `dashboard-template.js`, including its reading, watching, listening, empty, and undo surfaces. Both mount before dependent controllers query them, and focused tests cover location, idempotence, hidden defaults, status semantics, and all cross-media rail hooks. This removes another 51 lines from `index.html` without coupling the two features together.
- The Wiki/Notes feature now owns `wiki-template.js`, including its navigation, nested-folder form, search, trash, export, page tree, and workspace shell. `wiki.js` mounts it before binding drag/drop and editor behavior, with contract tests for every stable hook and its placement before the library surface. This removes another 45 lines from `index.html`.
- Music, podcasts, and audiobooks now own `audio-library-template.js`, separate from the already isolated player template. It contains browse/layout/sort controls, playlist creation, podcast discovery and subscriptions, bidirectional paging, virtualization spacers, and the list surface. Contract tests freeze all controller and accessibility hooks plus its placement before documents. This removes another 170 lines from `index.html`.
- Video and document catalogues now own separate `video-library-template.js` and `document-library-template.js` surfaces. Each controller mounts against the stable collection anchor, avoiding cross-feature construction dependencies while preserving video/audio/document order. Tests cover list, sort, pagination, hidden, and accessibility contracts. This removes another 31 lines and brings `index.html` below 1,000 lines.
- Appearance settings now belong to the theme feature through `theme-settings-template.js`. The theme controller mounts palette and brightness controls before general Settings binds its tabs, while retaining injectable controls for isolated tests. Contract tests cover all palettes, modes, explanatory copy, hidden defaults, placement, and idempotence. This removes another 32 lines from `index.html` and two composition-root lines.
- Playback settings now form a complete vertical module: `playback-settings.js` owns loading, dirty tracking, normalization, and serialization, while `playback-settings-template.js` owns its panel and language suggestions. `settings.js` delegates the entire slice and no longer knows its field selectors or defaults. Tests cover state round trips, normalization, mounting, DOM contracts, and preservation of unrelated settings; `index.html` drops another 44 lines.
- Metadata-provider credentials now form a focused vertical module: `metadata-provider-settings.js` owns saved-token state, library assignment status, connection testing, dirty tracking, and forward-compatible serialization, while its template owns the privacy and accessibility contract. Focused tests cover mounting, credible provider usage, exact test requests, token state, and preservation of future provider fields. This removes 60 lines from `settings.js` and 57 lines from `index.html`.
- Reading Sync now forms a complete vertical module: `reading-sync-settings.js` owns provider-dependent state, public/server address guidance, credential handling, paired-plugin delivery, Android build availability, dirty tracking, and forward-compatible serialization; its template owns the KoSync, plugin, and native-app setup surfaces. Focused tests cover normalization, dependent visibility, saved-secret handling, mount order, app availability, accessibility hooks, and unknown-field preservation. This removes 95 lines from `settings.js` and 138 lines from `index.html`.
- Storage settings now form a focused configuration module: `storage-settings.js` owns scan cadence, cache sizing/default retention, managed-ingest enablement validation, dirty tracking, and non-destructive serialization, while `storage-settings-template.js` owns the complete Storage panel. The transactional ingest workflow remains in its existing specialized controller instead of being folded into a broader settings monolith. Focused tests cover units, dependent validation, safety copy, mount order, and preservation of unrelated cache policy fields. This removes 40 lines from `settings.js` and 89 lines from `index.html`.
- Library administration now lives in `library-settings.js`: it owns new-library defaults, nested field commits, source/content compatibility, preferred-format migration, focused card rendering, deletion confirmation, and per-library scan polling. The global Settings coordinator delegates through `load`, `bind`, and `commit`, shrinking from 515 to 224 lines. Adding or removing a shelf now rerenders only library cards instead of resetting unrelated unsaved credentials across Settings. Focused tests cover independent defaults, format normalization, kind transitions, and storage compatibility; existing template contracts continue to freeze every form hook.
- Metadata review and managed-ingest workflow markup now lives in `metadata-review-template.js` and is mounted by `metadata-review.js` before Settings constructs its provider and ingest subcontrollers. Contract tests freeze every inspection, filtering, paging, batch-review, transaction-confirmation, status, and accessibility hook plus idempotent placement. This removes 194 lines from `index.html`, bringing the 335-line file to a cohesive app-shell role and removing it from the P0 list.
- Video playback now lives in `video-player.js`, which owns the player template, HLS lifecycle, bounded resume, preview frames, audio/subtitle selection, progress synchronization, fullscreen orientation, and teardown behind one `open` method. `video-library.js` retains catalogue, show, season, preference, and Continue Watching ownership and shrinks from 897 to 535 lines, removing it from the P1 list. Focused tests cover encoded HLS routes and resume boundaries; the existing player-template, HLS-loader, video-format, and progress-sync suites continue to cover its collaborators.
- Queue, favorites, and playlist HTTP behavior now lives in `audio_collection_http.go` as one user-owned music-collection boundary. Playlist collection and item routes are separate handlers and share a single music-entry validator. `audio_http.go` retains catalogue, streaming, artwork, analysis, and progress ownership and shrinks from 954 to 729 lines, removing it from the P1 list without changing routes or response contracts.
- Streamed playlist archives live in `audio_collection_download.go`, separate
  from collection mutation routes. The same boundary now serves albums through
  opaque group identities. It owns preflight validation, item/byte
  limits, safe archive names, constant-memory sequential ZIP delivery, stored
  audio entries, request cancellation, and the server-wide concurrency cap.
- Publication browsing now lives in `internal/query/books.go`: book filtering, title/recent ordering, cursor pagination, alphabet navigation, and indexed position lookup share one query boundary. `service.go` retains index lifecycle plus series/author projections and shrinks from 833 to 611 lines, removing it from the P1 list while the public `Service` API stays unchanged.
- The KOReader client is now a first-class embedded asset at `internal/web/koreader_plugin/main.lua`, with ZIP assembly isolated in `koreader_plugin_package.go`. Pairing persistence, catalogue matching, and the authenticated sync protocol remain in `koreader_plugin.go`, which shrinks from 942 to 369 lines and leaves the P1 list. Existing archive-contract tests continue to inspect the packaged Lua for text anchors, native locators, annotations, retry safety, and menu behavior.
- JavaScript assets now revalidate with ETags instead of remaining immutable for a year. This prevents a new composition root from loading an old dependency interface after deployment while retaining cheap `304` responses for unchanged modules.
- The accidental 9,199-line formatting expansion of `app.css` was corrected; the delivered shared file is compact again. Its 146 KB feature breadth remains a tracked P0 extraction rather than being hidden by the line count.
- Music Discovery now owns its complete responsive layout in `music-discovery.css` instead of keeping only three overflow safeguards there. The horizontal destination rail, two-column track layout, compact headers, artwork, typography, and mobile adaptation moved together without selector or value changes. This removes 1.6 KB from `app.css`; its reduced 144,464-byte budget is frozen while the remaining feature families are extracted.
- Fullscreen video playback now owns `video-player.css`, including dialog sizing, heading and track controls, media canvas, seek previews, mobile safe-area placement, and native-control adjustments. The file loads directly after shared tokens and primitives, while ownership tests prevent player selectors returning to the global bundle. This removes another 3.4 KB and freezes `app.css` at 137,321 bytes.
- Library scan feedback now owns `activity-panel.css`: the masthead indicator, queued/running/failed states, compact progress rows, fixed safe-area panel, scan-all control, and reduced-motion behavior move together. Ownership coverage prevents the monitor from leaking back into shared CSS, which falls another 2.2 KB to 135,171 bytes.
- The persistent audio player now owns `audio-player.css`: compact and expanded layouts, artwork transport, seek rails, timing, favorites, playback options, safe-area placement, chapter navigation, and reduced-motion behavior moved as one complete lifecycle. Mixed mobile rules were split so catalogue and dashboard selectors remain with their actual owners. The shared stylesheet drops by another 14.7 KB to 117,614 bytes; the shell gains one explicit stylesheet link, taking `index.html` from 335 to 336 lines without adding component markup.
- Notes/Wiki now owns `wiki.css`: its split workspace, nested navigation tree, drag/drop states, inline folder creation, Markdown reading/editor surfaces, backlinks, revision comparison, history panel, and phone adaptation no longer share a global stylesheet. Tree disclosure marks are drawn from the interface's 1.75px line grammar instead of Unicode glyphs. This removes another 11.3 KB from `app.css`, now 106,317 bytes; the shell's only growth is the explicit stylesheet dependency at 337 lines.
- Documents now owns `document-library.css`: its sticky sort control, cover and list catalogue modes, document actions, generated paper treatment, office-text flow, Markdown typography, reader toolbar, errors, and phone adaptation form one bounded 68-line stylesheet. The shared `app.css` falls to 98,300 bytes, below the ordinary 100 KB production ceiling, so its grandfathered exception and P0 finding are removed. `index.html` remains a 338-line composition shell whose one-line growth is the explicit feature dependency.
- Web-reader position persistence now lives in `reader-position-sync.js`, a 170-line state owner for revisioned loads and saves, exact-position coalescing, local pending state, retry backoff, online recovery, keepalive flushes, cross-device conflict blocking, remote acceptance, and lifecycle cleanup. `reader.js` delegates through a narrow facade instead of sharing eleven synchronization globals and shrinks from 712 to 615 lines. Focused tests cover coalescing with text anchors, revision advancement, conflict blocking, remote acceptance without echo writes, and absent server state.
- Reading-position equivalence is now authoritative in `internal/reading`: a
  stale reflowable save within the shared 0.2% renderer tolerance returns the current
  record as an idempotent success without mutating history or richer locators;
  PDF and comic positions instead require the exact page.
  The web synchronization owner accepts that response even when renderer CFIs
  differ, while materially different positions retain explicit conflict
  behavior. Store, HTTP, and browser regressions freeze the complete contract.
- A follow-up source audit corrected that last label: this web reader does not yet implement annotations, so there was no honest annotation controller to extract. The actual cohesive responsibility was cross-reader location translation. `reader-locator.js` now owns DOM text anchors, normalized EPUB text search, KOReader XPointer parsing, exact XPointer-to-CFI resolution, and safe chapter fallback. Exact, wrapper-drift, malformed, out-of-range, and missing-range tests cover the boundary. `reader.js` falls from 615 to 448 lines—below the preferred 500-line target—and leaves the P2 list while retaining only reader lifecycle, KoSync prompting/history, preferences, and controls.
- Position-history loading, safe rendering, restore confirmation, revision handoff,
  and recovery copy now live in `reader-position-history.js`. This restores the
  frozen reader composition budget after later synchronization work grew it to
  496 lines: `reader.js` is 430 lines and the focused history owner is 71.
  Empty and failed-history tests cover the extracted boundary.
- Android EPUB annotations now have an `EpubAnnotationSession` above the durable controller. It owns observable local state, explicit remote binding, device identity, export availability/in-flight state, stable-locator repair, local mutations, synchronization, Notes export, and theme-aware Readium decorations. The Activity no longer duplicates six annotation/session fields or manually reconstructs decorations, while focused tests cover state propagation, durable edits/deletion, and the requirement for complete credentials plus library/book identity. `EpubReaderActivity.kt` shrinks from 666 to 638 lines; it remains P2 because publication opening and custom-font runtime ownership are still mixed into lifecycle orchestration.
- Android EPUB custom-font runtime behavior now lives in `EpubFontRuntime`: stored selections are validated against installed fonts, Readium family names are derived consistently, and bounded font bytes are converted into one reusable browser-injection script. Tests freeze TrueType/OpenType MIME and format handling plus stable family identity. The Activity no longer builds CSS or Base64 payloads and shrinks from 638 to 623 lines; publication opening and locator observation remain the next cohesive boundaries.
- Android EPUB retrieval and parsing now live in `EpubPublicationLoader`, which owns Readium HTTP, asset-retriever, parser, and opener construction and returns explicit retrieval-versus-parsing failures. Focused tests freeze the two actionable user outcomes. The Activity no longer knows the Readium publication-loading stack and shrinks from 623 to 614 lines; locator observation and selection-action integration remain its largest mixed responsibilities.
- Android EPUB locator observation now lives in `EpubLocatorObserver`: lifecycle restarts, exact local position persistence, debounced reading statistics, structural cross-reader anchors, remote progress saves, error fallback, and resource-change repaint callbacks share one 103-line owner. The Activity receives only immediate, settled, synchronization, and repaint events and shrinks from 614 to 575 lines. Existing progress-coordinator, KOReader XPointer, locator-store, and reading-statistics suites continue to cover the collaborators; selection actions and reader-surface construction remain next.
- Android EPUB selection actions now live in `EpubSelectionActionController`: Android action-mode integration, duplicate-system-action avoidance, existing-highlight removal, stable context-scored anchor resolution, annotation creation, two-frame Readium repaint settling, diagnostics, decoration, and synchronization share one bounded owner. Candidate tests freeze same-resource preference, the safe single-result fallback, and ambiguous-match refusal. The Activity shrinks from 575 to 509 lines; reader-surface construction is the final substantial mixed responsibility before reassessing it as a composition root.
- Android EPUB view construction now lives in `EpubReaderSurface`, which owns generated container identity, reader/progress/chrome layer order, lifecycle-bound Compose disposal, chrome visibility, and the polite accessibility announcement host. The Activity supplies stateful feature content and callbacks but no longer manages raw view hosts or layering, shrinking from 509 to 486 lines. A responsibility audit now classifies it as a bounded composition root: it coordinates Android lifecycle, Readium fragment installation, and the already-focused font, annotation, selection, locator, progress, search, and surface owners. It therefore leaves the active P2 list instead of being split further solely to reduce line count.
- Local EPUB metadata parsing now lives in `internal/catalog/epub_metadata.go`: ZIP container lookup, bounded XML decoding, OPF models, creator roles and sort names, EPUB 3 collections, Calibre fallback, and shared publication-field normalization form one focused boundary. `catalog.go` retains storage, scanning, cache publication, acquisition, identifiers, and catalogue sorting and shrinks from 666 to 483 lines. An integration regression proves EPUB 3 series metadata wins over conflicting legacy metadata while non-author creators are excluded and refined author sort names remain intact.
- Video catalogue HTTP now lives in `video_catalogue_http.go`: video/season/show response models, bounded filtering and paging, show aggregation, progress summaries, metadata overrides, artwork and HLS/direct-play URL construction, and local/remote streaming share one route-family boundary. `catalogue_http.go` retains shared library discovery plus publication/author/series/synchronization routes and shrinks from 686 to 384 lines. Existing video HTTP tests continue to freeze show grouping, season artwork, episode responses, metadata overrides, progress, and route behavior across the extracted boundary.
- Managed-ingest filesystem operations now live in `internal/ingest/filesystem.go`: source symlink resolution and root confinement, regular-file and size validation, cancellation-aware bounded copies, SHA-256 verification, symlink-safe managed-directory creation, atomic no-clobber hard-link commits, directory fsync, and recovery observations share one security-sensitive boundary. `executor.go` retains planning, journal transitions, execution, recovery policy, rollback decisions, and retention and shrinks from 634 to 428 lines. Existing race-enabled traversal, symlink, size-change, collision, rollback, interrupted-copy, digest-mismatch, and manual-review tests continue to exercise the extracted primitives through the transaction API.
- Binary audio delivery now lives in `audio_delivery_http.go`: direct byte-range/download responses, artwork variants and sibling fallback, and on-demand HLS preparation/segments share one media-I/O route boundary. `audio_http.go` retains catalogue navigation, podcast projection, filtering/sorting, analysis dispatch, and progress ownership and shrinks from 729 to 543 lines. Focused contracts now cover range bodies and headers, download disposition, HEAD behavior, unsupported methods, artwork method rejection, and the existing end-to-end HLS playlist/segment flow.
- Missing music artwork now has one small presentation owner in `music-artwork-fallback.js`. Track rows and grouped album/artist views share its deterministic, Unicode-aware monogram instead of independently inventing placeholders. Artwork delivery remains backend-owned: cached attached-cover facts prioritize the bounded sibling search without moving media probing into UI controllers.
- Wiki Markdown rendering now lives in `wiki-markdown.js`. The bounded dialect,
  HTML escaping, Wiki-link attributes, fenced code, headings, emphasis, quotes,
  lists, empty content, and malformed persisted values are independently tested
  without constructing the full Wiki controller. `wiki.js` no longer mixes
  parsing with navigation, editing, history, trash, and drag/drop ownership and
  falls from 12,632 to 11,648 bytes. This is the first responsibility-based
  extraction from that compressed controller; future splits remain evidence-led
  rather than targeting its misleading 44-line count.
- Wiki page presentation now lives in `wiki-page-view.js`: active/deleted status,
  retention availability, action markup, rendered Markdown, and backlinks share
  one pure boundary. Malformed timestamps produce actionable fallback copy
  instead of `Invalid Date`, and backlink IDs are escaped before entering DOM
  attributes. `wiki.js` falls again from 11,648 to 10,439 bytes and retains
  state transitions plus event orchestration. Focused hostile-value and missing-
  data tests bring the frontend suite to 374 passing tests.
- Wiki revision-history presentation now lives in `wiki-history-view.js`.
  History rows, bounded revision values, timestamp fallbacks, escaped line diffs,
  unchanged-text explanation, and oversized-diff fallback no longer share the
  mutation coordinator. Restore requests remain in `wiki.js`, which falls from
  10,439 to 9,184 bytes. Malformed API responses and hostile snapshot content
  are covered directly; all 378 frontend tests pass.
- Wiki editor presentation now lives in `wiki-editor-view.js`, while shared
  rejection normalization lives in `wiki-errors.js`. Persisted title, folder,
  and content values remain escaped with their existing constraints and live
  status semantics. Save, move, purge, and folder-creation failures no longer
  assume every rejection has a string `message`; malformed failures retain a
  useful recovery path. `wiki.js` has converged on its cohesive role of state
  transitions, HTTP mutations, and event wiring at roughly 8.5 KB, so further
  splitting is intentionally stopped. All 384 frontend tests pass.
- Music Discovery now preserves partial success when one bounded catalogue
  request fails. Its loader publishes explicit per-section failure state rather
  than collapsing transport errors into empty arrays; the renderer owns honest
  empty copy and an in-place retry while healthy recent, album, artist, and
  playlist sections remain usable. Focused tests freeze stale-response safety,
  partial failure, recovery affordance, and the empty-versus-error distinction.
- Spoken-media fallback artwork and navigation chrome now begin a dedicated
  `audio-library.css` presentation boundary. Podcast and audiobook views use the
  same accessible SVG icon language as the rest of the shell instead of text
  glyphs, while `audio-media-views.js` remains a pure renderer and `app.css`
  receives no new feature rules.
- Series and author projections now live in `internal/query/projections.go`: grouping, representative selection, cached keyset pages, filtering, sorting, and pagination form one read-model boundary. `service.go` retains index construction, version invalidation, and immutable snapshot publication and shrinks from 611 to 197 lines; book filters and comparators stay with `books.go`. Projection pages now copy cached slices before returning them so a caller cannot mutate shared index state, with a focused regression covering both series and authors.
- Server runtime initialization now lives in `multi_runtime.go`: persistent account/progress/annotation/collection stores, KoSync setup, metadata-override wrapping, library validation, OPDS handler construction, query indexing, and health wiring are grouped into three failure-returning initialization stages. `multi.go` remains the public configuration/type and HTTP composition root and shrinks from 580 to 394 lines. Table-driven startup contracts preserve empty-library, partial-credential, invalid-library, missing-store, and duplicate-ID errors, while existing routing tests verify the initialized handlers.
- Jellyfin catalogue compatibility now lives in `jellyfin_catalogue_http.go`: library, series, season, video, and audio identity lookup plus browse/item/artwork responses share one projection boundary. `jellyfin_http.go` retains authentication, protocol dispatch, playback negotiation, direct streaming, and progress reporting and shrinks from 531 to 307 lines. The local video stream path now performs one catalogue lookup instead of two full scans per request; the race-enabled compatibility suite covers login, hierarchy browsing, local and remote range delivery, account-bound progress, and disabled-account rejection across the split.
- The podcast store remains a cohesive persistence boundary rather than being split solely at 482 lines. Its feed accessors now deep-copy episode slices so API and downloader callers cannot mutate or race persisted state through shallow snapshots. Aggregate episode snapshots copy under the read lock but sort after releasing it, reducing lock hold time from O(n log n) to O(n) for large feeds; focused race coverage freezes snapshot isolation.
- Immutable snapshot ownership is now consistent across local/Seafile books, audio, video, and reading state. Shared clone helpers deep-copy author lists, optional series indexes, chapters, media streams, alternate KoSync documents, web anchors, and locator maps at API boundaries. Focused race regressions mutate every returned shape and verify stored state remains unchanged. This work also exposed and fixed an older `clonePosition` defect that replaced locator maps before iterating them, which could erase client-specific Readium/Foliate locators during rollback cloning.
- Web audio progress is now owned by `audio-progress.js`: device identity,
  revisioned saves, timed-save throttling, active conflict retry, and passive
  refresh behavior no longer share mutable synchronization fields with the
  catalogue coordinator. `audio-artwork-loader.js` separately owns
  viewport-aware observation, a four-request image queue, successful-load
  transitions, failed-image removal, and cleanup for virtualized nodes. Focused
  tests cover both boundaries. Album/artist catalogue state has since moved to
  the cohesive owner described below; `audio-library.js` still coordinates the
  distinct podcast, audiobook, ordinary-track, and player surfaces, so future
  extractions must continue to follow those responsibilities rather than line
  count alone.
- Web audio progress now also owns passive-device write relinquishment. An
  inactive player that discovers a newer revision refreshes its base but blocks
  later lifecycle writes from replaying the stale position; only an explicit
  Play event or item reset restores write ownership. The focused regression
  covers the conflict, repeated forced save, and deliberate ownership reclaim.
- Web audio transport now closes its browser/operating-system media-session
  projection together with the visible player and both playback decks. A
  focused transport regression prevents stale metadata or a resumable
  `playing` state from surviving an explicit Close action.
- Web video now relinquishes its active catalogue item immediately after its
  explicit final checkpoint and before `pause` dispatches. Closing the player
  therefore cannot enqueue the same still-owned checkpoint twice or leave a
  stale item reachable by delayed media callbacks; a focused lifecycle
  regression freezes the ordering.
- Android passive audio and video writes now share the bounded pure policy in
  `ProgressOwnershipPolicy.kt`. It prevents stale clients from rewinding or
  unfinishing newer remote state and centralizes the server-aligned 98% audio
  completion threshold, avoiding divergent policy hidden in two HTTP clients.
- Android's server-derived audio resume seek is classified by the focused,
  Android-free `AudioSeekOrigin.kt` policy. Its one-shot token cannot suppress a
  later real scrub, keeping Media3 event handling out of progress ownership.
- TV Now Playing keeps its long queue virtualized while deriving the initial
  viewport through the pure `tvQueueInitialIndex` boundary. The queue seeds one
  route-scoped focus identity for the active track; subsequent edits remain
  owned by the existing bounded `TvGridPositionStore` rather than a new screen
  controller.
- Android mobile root navigation is now a single `MobileShellDestination`
  state in `AppShell.kt`; Settings no longer bypasses the route model through a
  second Boolean. A focused pure policy test freezes system-Back ownership for
  every non-Home root without coupling navigation to any feature ViewModel.
- Web music destination gestures remain inside `music-destinations.js`, beside
  keyboard and active-rail synchronization. The audio coordinator supplies one
  gesture surface but does not absorb thresholds or input exclusions; focused
  tests freeze ordered movement, bounded edges, and short-gesture rejection.
- Web playlist presentation now lives in `music-playlist-view.js`: compact root
  cards, representative artwork, focused detail hierarchy, empty states, and
  guarded destructive controls form one escaped rendering boundary. Playlist
  ordering policy remains a pure tested helper in `music-state.js`; the audio
  coordinator retains only the selected playlist name and route events.
- HLS disk-cache inspection and eviction now live in `internal/video/hls_cache.go`.
  Contiguous segment discovery, completed-playlist detection, bounded directory
  accounting, LRU ordering, and protection of active/recent sessions form one
  storage-lifecycle boundary. `hls.go` retains quality profiles, encoder/session
  orchestration, seek restarts, job ownership, media opening, and manifest
  generation; cache tests moved beside their owner and the race-enabled video
  suite covers the unchanged concurrency contract.
- Atom/OPDS and OpenSearch serialization now lives in
  `internal/web/opds_wire.go`. Shared feed models, media types, navigation-entry
  identities, newest-update calculation, XML escaping, namespace selection,
  and response encoding form one wire-format boundary used by both catalogue
  modes. `web.go` retains single-library routing, filtering, acquisition, page
  streaming, and authentication and shrinks from 493 to 411 lines. Focused
  response tests freeze timestamps, PSE namespaces, media types, XML decoding,
  and configured-URL escaping.
- Fine-pointer playlist and queue dragging now lives in
  `music-order-drag.js`, which owns reusable delegated drag lifecycle and
  drop-target presentation for an explicit row selector. The pure,
  independently tested ordering policy remains in `music-state.js`, while the
  audio coordinator only persists a completed move. Existing menu actions stay
  the accessible keyboard and touch path.
- Web music metadata presentation now lives in `music-track-info.js` with its
  own `music-track-info.css`: field validation/formatting, escaped dialog
  rendering, modal lifecycle, artwork, and responsive presentation form one
  bounded feature. `audio-item-view.js` contributes only the Track info action,
  and the audio coordinator contributes only the selected item.
- Web track-collection presentation now lives in
  `music-track-collection.js`. Favorites and Recently Played share one pure,
  tested cover/list boundary instead of duplicating conditional markup inside
  the audio coordinator. Album identity and its Play/Shuffle controls remain in
  `music-group-view.js`; `audio-library.js` owns only the resulting playback
  events and queue state.
- Web Artist → Album → Track request state now lives in
  `music-group-catalogue.js`. It owns hierarchy selection, group pagination,
  alphabet navigation, cancellation, and stale-response rejection while the
  coordinator retains DOM rendering and playback. A missing next-offset header
  is represented as the end of pagination rather than numeric offset zero,
  preventing accidental first-page reloads. Focused tests cover root paging,
  artist drill-down, album tracks, latest-request wins, and navigation reset.
- Web audiobook catalogue state now lives in `audiobook-catalogue.js`. Series
  selection, series loading, book paging, cursor ownership, index-change retry,
  request cancellation, and stale-response rejection no longer share the
  coordinator's ordinary audio request slot. Focused tests cover hierarchy,
  cursor paging, newest-request ownership, and the bounded 409 restart;
  `audio-library.js` falls from 43,838 to 43,049 bytes and its budget is frozen
  at 43,100 bytes so this responsibility cannot drift back into it.
- Web podcast catalogue state now lives in `podcast-catalogue.js`. Selected-show
  ownership, episode filters, cursor paging, cancellation, stale-response
  rejection, and bounded index-change recovery no longer share the ordinary
  audio request slot. The extraction also fixes continuous paging for an open
  podcast, which previously called the generic audio loader instead of loading
  the next episode page. Focused tests cover filters, cursor use, latest-request
  ownership, recovery, and closing a show; `audio-library.js` falls again from
  43,049 to 42,603 bytes and its frozen budget tightens to 42,650 bytes.
- Ordinary web audio transport now lives in `audio-root-catalogue.js`. Music
  tracks and spoken-media root pages share one tested owner for encoded query
  construction, cursor/offset selection, page-size policy, abort propagation,
  and defensive pagination-header parsing. The UI coordinator retains stale-
  request ownership, bounded DOM-window updates, navigation settling, and
  rendering; its byte ceiling tightens to 31,600 so transport cannot drift back
  into presentation state.
- Web audio browse preference persistence now lives in
  `audio-view-preferences.js`. It owns sort/layout restoration, per-media sort
  keys, storage writes, and closing the shared View menu; catalogue reload
  decisions remain in `audio-library.js`. Focused tests cover defaults,
  persisted layouts, menu dismissal, and isolation of podcast/music sort keys.
- Home's Continue Listening rail now lives in `audio-continue-rail.js`. It owns
  spoken-media filtering, the four-item bound, rendering, dismissal, request
  cancellation, and stale-response rejection instead of extending the audio
  catalogue coordinator. Focused tests freeze the visibility and completion
  policy; `audio-library.js` falls from 39,098 to 38,465 bytes.
- Web playlist interaction state now lives in `music-playlist-controller.js`.
  Playlist selection, creation, rename, confirmed deletion, playback, and track
  membership no longer share the general audio catalogue event lifecycle.
  Focused tests cover selection, queue-before-play ordering, cancelled deletion,
  and idempotent membership; `audio-library.js` falls again to 35,791 bytes.
- Compact web dashboards now treat Continue Listening as a true horizontal
  media rail. Each spoken-media card retains side-by-side artwork, bounded
  two-line description copy, and a wide readable viewport instead of collapsing
  into stacked half-width content. Live browser checks enforce card width,
  artwork size, and the two-column internal layout at 390 px and 320 px.
- Web Music Discover now applies progressive disclosure to Favorites. The
  dedicated destination remains permanently available, while its dashboard
  shelf is omitted until at least one favorite exists; this moves album artwork
  into the first desktop viewport without adding density or administrative UI.
- Android TV library composition now crosses a typed contract boundary in
  `TvLibraryContracts.kt`. Artwork, navigation, publication, audio, video, and
  podcast actions are grouped by responsibility instead of passed through a
  fragile positional callback list. Mobile rail selection and reveal behavior
  is likewise isolated and tested: Home occupies its own index, only one
  destination is selected, and an off-screen active shelf scrolls into view.
- Android TV content-route focus identity includes both the selected library
  and its open hierarchy. Switching between two root shelves now moves focus
  to the stable edge rail instead of leaving it attached to a card removed from
  composition; paging, filtering, and ordinary recomposition remain excluded
  so they cannot steal focus during browsing.
- Web audio playback ordering now lives in `audio-playback-order.js`. It owns
  stable shuffle construction, active-track preservation, queue-mutation
  revalidation, adjacency, and bounded queue edges. Deterministic tests cover
  shuffled and ordinary playback, leaving the audio catalogue coordinator with
  less transport state while keeping the boundary independent of DOM rendering.
- Android passive playback synchronization now has focused transport/policy
  owners. `AudioProgressClient` retains a bounded revision cache and avoids a
  preflight GET on every timed save; `VideoProgressClient` owns revision-aware
  reconciliation instead of leaving raw unconditional HTTP writes in
  `VideoPlayerActivity`. Both refuse to overwrite newer or finished remote
  state and retry a race once only when local playback is genuinely ahead.
  Web and Android now share a 250 ms clock-equivalence window: independently
  sampled player clocks do not manufacture conflicts, while a stale client can
  no longer classify a multi-second rewind as harmless drift.
- Durable Android audio outbox entries now timestamp explicit seek intent and
  retain it for at most ten minutes. A pause immediately following an offline
  rewind still carries that intent, but an entry replayed after a long process
  or network outage is downgraded to passive progress and cannot rewind newer
  playback from another device. Legacy untimestamped entries fail safe as
  passive writes.
- EPUB startup reconciliation now distinguishes a clean same-device position
  from a pending local locator created while offline. Clean sessions still
  resume the acknowledged server locator automatically; pending offline reading
  remains local, adopts the current server revision, and is uploaded by the
  existing movement observer instead of being replaced by an older server page.
  Divergent positions from another device still require explicit review.
- PDF and CBZ exact-page persistence now share
  `FixedLayoutLocalProgressStore`. Settled pages still use inexpensive
  asynchronous writes while reading, but both reader activities synchronously
  commit the latest observed page during `onPause`, matching EPUB's lifecycle
  durability guarantee. Existing `pdf_reader`, `comic_reader`, and `page:` keys
  remain unchanged, so upgrades retain every local position.
- EPUB progress presentation now lives in `EpubReadingProgressChrome.kt`.
  Persistent percentage/chapter markers, learned chapter and book time labels,
  the transient progress badge, and the statistics panel no longer share the
  menu/navigation implementation in `EpubReaderChrome.kt`. Existing pure tests
  continue to own footer visibility and label policy, while the architecture
  gate prevents these independent responsibilities from being merged again.
- Android video recovery now distinguishes automatic restoration from viewer
  intent through `VideoSeekOrigin`. Media3 reports reconnect, fallback, retry,
  and scrub movements with the same seek discontinuity reason; matching
  recovery seeks are therefore saved as passive progress and remain subject to
  monotonic conflict protection, while a deliberate viewer seek may still
  rewind or reopen completed media. Focused tests cover matching, one-shot,
  mismatched, and cleared expectations.
- Browser video now applies the same ownership distinction through the focused
  `media-seek-origin.js` policy. Direct-play resume, HLS resume, and audio-track
  replacement seeks remain passive; a deliberate native-control scrub carries
  one `explicit_seek` claim through revision reconciliation. This lets the
  viewer intentionally rewind without allowing an automatic reconnect or stale
  tab to overwrite newer progress.
- Android video failure recovery now lives in `VideoPlaybackRecovery.kt`.
  Network errors and buffering stalls share one bounded retry budget, a
  one-shot compatibility fallback, manual-retry reset, and identity-aware media
  transitions. Replacing an episode with its fallback stream can no longer
  reset fallback ownership and loop. Transition tests cover transient,
  permanent, mixed stall/error, manual retry, same-item replacement, and true
  episode changes. `VideoPlayerActivity.kt` retains ExoPlayer lifecycle, UI,
  queue transitions, progress, tracks, picture-in-picture, and scrub previews;
  it shrinks from 24,871 to 24,174 bytes and its 486-line composition boundary
  is frozen against renewed growth.
- Android reader defaults and hardware-navigation presentation now live in
  `ReaderDeviceSettings.kt`. Font discovery/import, private-font preview,
  typography defaults, and BOOX-style volume-key paging form one reader-owned
  settings boundary; `DeviceSettingsScreen.kt` retains category routing,
  connection, appearance, libraries, downloads, playback, privacy, and dialogs
  and shrinks from 399 to 310 lines.
- Android mobile playback presentation now lives in `MobilePlaybackHost.kt`.
  Its mini-player and expanded-sheet restart scopes alone collect the frequently
  changing `PlaybackState`; `MochilApp` no longer observes the one-second
  clock while composing every feature binding. `PlaybackViewModel` exposes a
  distinct ReplayGain state for Settings, preserving immediate preference
  updates without reintroducing clock-driven Settings recomposition.
- Android mobile queue and playlist subscriptions now follow the same boundary.
  The connected composition root no longer observes either complete state;
  Library collects playlists only while that destination exists, and the
  expanded-player route returns before subscribing when its sheet is closed.
  Queue edits and playlist refreshes therefore cannot invalidate Home,
  Settings, or the navigation shell, while the mini-player remains independent.
- Library directory observation now has shared `LibraryShellProjection` and
  `LibraryVisibilityProjection` boundaries. Mobile Home, the mobile shell, TV
  Home, and both Settings surfaces collect only their distinct directory slice;
  book paging, search, group loading, reader opening, and collection state can
  no longer recompose unrelated navigation or administrative screens.
- Final Android audio positions now pass through `AudioProgressJournal`, a
  bounded encrypted outbox separate from `PlaybackService` orchestration and
  `AudioProgressClient` transport. Task removal cannot cancel the only durable
  copy: acknowledged entries are removed conditionally so an older request can
  never erase a newer queued position for the same item.
- PDF and CBZ readers now share a durable `PdfProgressStore` checkpoint that
  records the server revision represented by the exact local page. Opening an
  untouched copy follows a newer device automatically, whereas locally changed
  offline progress still requires an explicit choice. Transport remains in
  `PdfProgressClient`, orchestration remains in `PdfProgressViewModel`, and the
  pure open-position policy has focused regression tests.
- Fixed-layout progress now survives failed writes as a durable one-item outbox
  per publication. PDF and comic locators use distinct wire keys, comic reads
  retain compatibility with the former PDF key, and explicit library/item IDs
  prevent colon-containing identifiers from being reparsed incorrectly. Web
  EPUB/CBZ opening now applies the same clean-remote-versus-pending-local policy,
  understands exact comic pages, periodically offers newer Mochil positions,
  and gives an explicit Stay-here action fresh write ownership. Focused Android
  and browser tests cover offline replay, newer queued pages, exact locators,
  remote acceptance without echo writes, and deliberate local reclaim.
- Fixed-layout synchronized state now uses the same unambiguous identity
  discipline as its network routes. Length-prefixed library and publication IDs
  prevent colon-containing pairs from aliasing another PDF/CBZ; the first
  connected open atomically migrates legacy revision, acknowledged-page, and
  pending-page keys. A collision regression and the architecture gate freeze
  the new identity boundary.
- Exact PDF/CBZ device state now follows that boundary too. Reader activities
  migrate the legacy page and finished flags before opening, so reopening a
  publication remains exact without allowing colon-containing IDs to alias a
  different book.
- Android audio now has one length-prefixed `AudioTrack` identity shared by
  favorites, queues, downloads, podcast recovery, playback lookup, and Compose
  item keys. Existing download-index entries normalize through their embedded
  track metadata, preserving files while removing the former delimiter
  ambiguity. Collision and migration regressions plus the architecture gate
  freeze this contract.
- Video queue entries now use the same length-prefixed discipline for Media3
  item identity and reconnect ownership. This state is session-local, so no
  persisted migration is required; a focused collision regression protects it.
- Compose catalogue identity now shares a small `CatalogueIdentity` boundary.
  Publication cards, series, authors, downloads, video grids, and TV Continue
  Watching cannot alias rows when a library or item identifier contains a
  delimiter; this prevents state and focus from being reused by the wrong card.
- Android Store hardening now has executable contracts rather than a checklist
  alone. The common manifest has only network and foreground-media permissions,
  reader/player activities remain private, the exported media session is guarded
  by the platform bind permission, notification permission is unnecessary for
  the exempt media-session notification, and only the direct flavor can request
  package installation. Device credentials remain Android-Keystore encrypted,
  login binds credentials to the exact user-approved HTTPS origin, malformed
  credentials are rejected, and every bearer-authenticated `HttpURLConnection`
  disables automatic redirects. The latter is source-scanned by unit tests so a
  future API client cannot silently forward a device token to another origin.
  The store-readiness gate additionally freezes disabled backup/device transfer,
  disabled cleartext traffic, a private FileProvider, and the absence of any
  personal deployment hostname from distributable source and packaged APKs.
  Self-hosted addresses therefore remain runtime configuration rather than an
  accidental dependency on the developer's installation.
  The same gate now freezes the Android project graph to the single
  distributable `:app` module and rejects retired integration-module
  directories. Dormant experimental code therefore cannot quietly restore its
  dependency graph merely by living outside `app/src`.
- Android TV remote holds now have an explicit input-ownership handoff between
  a card and its context popup. The popup consumes the opening Select release
  to prevent accidental activation, while the card immediately clears its hold
  state so cancelling the popup cannot swallow the next ordinary click. A pure
  state-machine regression covers this formerly intermittent focus failure.
- Android TV network recovery crosses the existing typed navigation boundary.
  One focused error surface invokes a media-specific retry action; publication
  retries select the smallest failed discovery, shelf, or group scope through a
  pure tested policy instead of duplicating recovery logic in each TV grid.
- Android collections now have an authenticated native-route integration test:
  it creates a collection through the device API, adds a series member, verifies
  the resolved count, and retrieves the expanded books through the same endpoint
  used by the app. Mobile publication alphabet selection also matches web and TV:
  selecting the active letter or `#` clears the filter instead of forcing an
  extra trip through another browse mode.
- Android audio presentation now shares one hour-aware playback clock across
  the mini-player, expanded timeline, audiobook chapters, bookmarks, and
  accessibility labels. Effective source identity is carried with each Media3
  item and its bounded playback snapshot, so reduced-quality HLS correctly says
  `AAC · 128 kbps · Converted` instead of misleadingly displaying the source
  file's FLAC facts. Direct playback retains its real container, bit depth, and
  sample rate. Shuffle is progressively disclosed only for music rather than
  occupying podcast and audiobook controls. Pure tests cover URI classification,
  restored source facts, short/long clocks, and media-kind control policy.
- Music Discovery's content-first contract is now checked against the rendered
  application as well as its pure renderer: the browser smoke suite asserts
  that the bounded Recently played shelf never regains a per-track progress
  line. This catches composition or asset-delivery regressions that isolated
  JavaScript tests cannot observe.
- Web loudness-analysis state now lives in `audio-analysis-controller.js`.
  It owns shelf eligibility, request/error presentation, confirmation, job
  polling, stale-response rejection, and timer cancellation when the user
  changes or leaves a shelf. Its controls live in the audio template, so
  `audio-library.js` no longer manufactures analysis markup or retains a
  background timer; the coordinator shrinks from 40,205 to 39,098 bytes and
  its frozen byte budget follows the smaller boundary.
- Android podcast device-download ownership now lives in
  `PodcastDownloadCoordinator.kt`. It contains local-file recovery, RSS/server
  identity matching, retention policy, unmetered-network policy, and guarded
  automatic downloads; `PodcastViewModel.kt` retains subscription, paging, and
  UI-state orchestration. The view model shrank from 453 lines / 26,643 bytes
  to 364 lines / 21,263 bytes. The module-size gate prevents those storage and
  network-policy responsibilities from drifting back into the view model.
- Android music transport now lives in `MusicCatalogueRepository.kt`. It owns
  blocking catalogue calls, parallel discovery queries, favorite writes, IO
  dispatch, and performance measurement; `MusicViewModel.kt` retains UI state,
  route generation, stale-response rejection, playback URLs, and download
  coordination. The view model shrank from 24,534 to 22,362 bytes, and the
  module gate prevents direct catalogue transport from returning to it.
- Web audio catalogue click routing now lives in
  `audio-library-interactions.js`. Ordinary item actions, queue capture, and
  Discovery navigation retain their deliberate propagation order without
  sharing the catalogue/loading coordinator. `audio-library.js` shrank from
  36,302 to 33,877 bytes, focused interaction tests cover the capture boundary,
  and the module gate prevents delegated DOM routing from drifting back.
- Podcast resume lookup now uses the focused `PodcastPlaybackResolution`
  generation owner. `PodcastViewModel` exposes only one resolving item in its
  presentation state, while mobile and TV routes cancel ownership when their
  library surface leaves composition. Slow cross-device lookups cannot outlive
  navigation or a newer play request; focused pure tests freeze newest-request
  and cancellation behavior without moving transport into Compose.
- TV live-queue presentation lives in `TvPlaybackQueue.kt`, separate from the
  Now Playing transport composition. `PlaybackViewModel` owns the Media3 queue
  snapshot and bounded play/move/remove operations; the TV route supplies only
  the persistence callback for music and audiobooks. This keeps virtualized
  queue rows and held-item menus out of the player lifecycle coordinator while
  avoiding a second queue model in Compose. The snapshot is also published as
  soon as Media3 reconnects, so a process-restored session does not present an
  empty TV queue while it waits for the next timeline event.
- Expanded web-player queue presentation lives in `audio-player-queue.js`.
  It owns bounded paging, active-row semantics, focus restoration, and panel
  events; `audio-library.js` supplies only current queue state and the existing
  playback callback. This keeps another stateful surface out of the audio
  coordinator while preventing a large synchronized queue from becoming a
  large player DOM.

## Extraction order

1. Finish and stabilize the vault reader modules.
2. Extract feature templates from `index.html`, starting with vault/documents and settings.
3. Split `app.js` routing and catalogue ownership without changing behavior.
4. Move remaining feature CSS out of `app.css` whenever its owning feature is touched.
5. Split the P1 backend and frontend modules with race/unit coverage around each boundary.

Every extraction must pass Go race tests, frontend tests, Android tests where applicable, and the design detector for affected UI surfaces.

The Android collection action follows this boundary policy: `CollectionClient`
owns transport, `CollectionViewModel` owns asynchronous picker state,
`CollectionModels` owns the wire/domain shapes, and `CollectionPickerSheet`
owns presentation. Long-press surfaces receive only callbacks and do not absorb
collection persistence or networking.

Android series browsing now lives in `PublicationSeries.kt`; the general
publication surface retains book browsing, details, filtering, and shared
primitives. This moved the previously growing `PublicationScreen.kt` from 511
to 429 lines while keeping series count badges and redundant-author policy in
the feature that owns them.

Android reader identity now lives in the focused `ReaderDeviceName.kt` helper.
EPUB/PDF transport supplies a bounded manufacturer/model label, while the EPUB
progress coordinator owns the divergent-progress explanation and receives the
complete remote position rather than presentation fragments. The Go reading
model and HTTP boundary independently reject oversized labels. Focused Android
flavor and Go tests freeze both the user-facing decision policy and the wire
limit without adding lifecycle responsibility to either reader Activity.

Responsive web destination styling now lives in `destination-rail.css` and is
attached by the existing `createDestinationRail` behavior owner. Publication,
music, and audiobook navigation therefore share one touch floor, keyboard
state, active underline, scroll snapping, and two-sided overflow cue while
their feature styles retain only placement and density adaptations. Ownership
tests reject feature-local active-state copies so the rails cannot drift back
into subtly different controls.

Android TV Now Playing focus transitions now live in the existing
`TvNowPlayingFocusSession`: transport receives focus on entry and after Queue
closes, while track changes and recomposition cannot steal it. The bounded
`TvGridPositionStore` is owned one level higher by `TvConnectedApp`, so replacing
the library route with Now Playing no longer destroys the remembered catalogue
row and item. Route and screen layers receive this narrow state owner instead of
duplicating navigation memory or growing another global view model.

Android video pagination stays behind the existing `VideoClient` and
`VideoViewModel` boundaries. `VideoClient` owns the bounded page protocol and
total-count response parsing; `VideoViewModel` owns generation-safe continuation
and independent show/item cursors; mobile and TV receive narrow load-more
callbacks. This prevents a two-pane show list and episode list from sharing
mutable paging state while avoiding a second video coordinator or screen-sized
transport implementation.

Web movie recycling reuses the shared `bounded-window.js` measurement owner
instead of adding height and scroll bookkeeping to `video-library.js`. The video
controller owns only movie offsets, retained domain items, and forward/backward
requests; the shared primitive owns DOM trimming, spacers, and anchor
preservation. Interactive show and season trees remain explicitly unwindowed.

Ordinary document recycling follows that same boundary: `document-library.js`
owns document cursors, offsets, and cursor-conflict recovery, while the shared
window owns measurements and DOM eviction. Vault trees bypass the flat-list
window deliberately, keeping their hierarchy, selection, and editing lifecycle
inside the existing vault owner rather than coupling it to pagination state.

Web audio control structure now belongs completely to the existing feature
templates. Podcast played filtering, the complete music destination select,
audiobook series sorting, and the audiobook chapter action are declared in
`audio-library-template.js` or `audio-player-template.js`; `audio-library.js`
only binds their state and behavior. A focused architecture regression rejects
those control fragments if they drift back into the catalogue coordinator,
without forbidding its intentional incremental insertion of paged track rows.
Web audio paging and bulk-action presentation now has a separate pure policy in
`audio-render-policy.js`. It calculates retained/available counts, finite music
destinations, load-more labels, and podcast/audiobook Play All availability;
`audio-library.js` applies the result without re-encoding the media rules in its
DOM renderer. Focused tests cover grouped music, recycled track offsets, podcast
shows, and partially loaded audiobook series.
Podcast and audiobook surface selection now stays with the existing
`audio-media-views.js` family owner. It chooses overview, focused group,
cover/list, filtered-feed, and empty-state markup; `audio-library.js` supplies
catalogue state and applies the result. This reduces the coordinator to 218
lines / 31,834 bytes without inventing a generic component abstraction, and
the architecture gate freezes both the family boundary and the smaller byte
budget.

Run `scripts/check-module-sizes.sh` during verification. Existing oversized production files have frozen budgets: they may shrink, but new feature work cannot make them larger. New production modules have an 800-line and 100 KB hard ceiling; the preferred boundary remains below 500 lines.

Native Wiki offline reopening is isolated in `WikiOfflineStore.kt`: bounded
snapshot encoding, decoding, encryption, and privacy-safe storage timing do not
expand the Wiki state coordinator. `WikiViewModel.kt` remains a 461-line state
owner, while the storage module is 150 lines. Android library-summary request
coalescing likewise lives in the 47-line `LibrarySummaryRequestCache.kt` rather
than adding process cache state to `LibraryViewModel`; focused concurrency and
expiry tests cover that boundary, and the module-size gate passes.
Wiki offline storage now also derives an opaque encrypted slot per vault and
keeps a four-vault LRU so total cache storage remains bounded.
Legacy single-slot data migrates on first use, and one vault can no longer
displace another vault's recent pages; identity and cross-vault merge
regressions protect the boundary without moving storage concerns into the
ViewModel.
Bounded vault-page acquisition lives in the separate 51-line
`WikiOfflineMirror.kt` owner. It owns Markdown target selection, four-request
batching, progress, and cache publication; Compose receives one action and
status state rather than network or storage responsibilities.

Android bounded text-response IO now lives in the 22-line
`network/BoundedResponseReader.kt` primitive. Authentication, catalogue,
collections, dashboard, audio, podcast, video, progress, annotation, Wiki, and
resume-metadata owners choose their domain ceiling but no longer duplicate or
bypass the byte-counting loop. Binary media and export streams remain with
their existing specialized bounded-copy owners.
`network/JsonBounds.kt` supplies the complementary item-count guard. Feature
clients retain their own contract-aligned limits, while the shared primitive
keeps overflow behavior consistent and tested without creating a transport
god-object.

Native Wiki compound remote operations now live in the focused
`WikiRepository.kt` boundary. Tree-plus-selection opening, refresh, authoritative
post-save reload, creation, move, trash, search, and backlinks no longer mix IO
sequencing into `WikiViewModel`; the ViewModel retains request cancellation and
presentation ownership. A narrow `WikiRemote` interface makes operation order
and partial-save failure behavior independently testable. In particular, a
successful write followed by a failed revision reload is no longer presented as
an ordinary failed save that encourages a duplicate retry. The ViewModel is 444
lines and the repository is 112 lines; the architecture gate prevents remote
sequencing from drifting back into the state owner.

Reading progress equivalence now has a semantic reflowable-location boundary.
The server acknowledges a stale CFI, KOReader XPointer, or contextual text
anchor when it names the authoritative sentence even if screen-dependent
fractions differ substantially; it never replaces newer revision, device,
locator, timestamp, or history state. Web save acknowledgement mirrors the same
policy in `reading-position-equivalence.js`, while PDF and CBZ remain exact-page
only. Ambiguous context-free anchors and different spine locations still
conflict.
