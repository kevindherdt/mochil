# Mochil stabilization and modular growth plan

This document records the approved order of work before Mochil expands into
video, television, podcasts, music, or integrated AI. The existing book
catalogue and reader remain production behavior; expansion must not weaken it.

## Engineering principles

- Canonical media stays in the configured storage backend. Mochil indexes,
  caches, serves, and records user state without silently reorganizing files.
- A storage backend exposes small optional capabilities such as ranged covers,
  pages, or streams. It must not know about HTTP routes or presentation.
- Published catalogue snapshots are immutable. Requests never observe a scan
  halfway through and derived indexes can be replaced atomically.
- Every expensive remote operation is bounded, cancellable, cached where safe,
  and isolated per library.
- Existing OPDS, reader, acquisition, and synchronization contracts receive
  regression tests before shared code is generalized.
- Persistent formats are versioned and migrated forward. Upgrades do not edit
  canonical library content.
- Mochi and the other applications under `otherapps` are behavior references,
  not codebases to merge. Specify and test useful behavior before writing a
  smaller Go-native implementation.

## Non-negotiable code-health requirements

- Packages and frontend modules own one cohesive responsibility. Storage,
  media interpretation, indexing, transport, presentation, and user state keep
  separate boundaries.
- HTTP handlers only validate input, invoke a service, and encode a response.
  They do not crawl storage, infer metadata, transcode, or implement sync rules.
- Storage adapters expose small capability interfaces. Adding another backend
  must not require changes throughout every media module.
- Books, comics, video, television, podcasts, music, documents, and AI remain
  independently removable modules. Disabling unfinished work restores existing
  application behavior.
- Shared models contain only genuinely shared fields. EPUB CFIs, comic pages,
  video seasons, podcast feeds, and music tags remain in their owning modules.
- Expensive work has bounded queues, concurrency, reads, decoded sizes, retries,
  temporary storage, and execution time, and is cancellable and observable.
- Catalogue snapshots are immutable after publication. Failed scans and
  configuration changes preserve the last known-good library.
- Persistent state is versioned, atomically written, forward-migrated, and
  bounded. Canonical media is never treated as disposable application state.
- New abstractions require at least two real consumers. Prefer small, clear
  duplication over a premature general-purpose framework.
- Split a file when it accumulates unrelated behavior. Length is a warning;
  cohesion, dependency direction, and testability determine the boundary.
- Go remains the server language. Browser JavaScript stays framework-light and
  is split by screen or feature as it grows.
- Every feature adds focused tests and preserves OPDS, acquisition, reading,
  KoSync, catalogue, cache, and metadata contracts with regression and race
  coverage where applicable.

## Change gate

Every phase or vertical slice must answer before completion:

1. Which package owns it, and which packages may depend on it?
2. Can it be disabled without changing books, comics, OPDS, or reading sync?
3. Which expensive operations are bounded, cached, cancellable, and observable?
4. What persistent state does it own, and how is it migrated and restored?
5. What happens when its provider, storage, or worker fails?
6. Which automated tests prove its contracts?

If those answers are unclear, implementation pauses at the design boundary.

## Current implementation status

- The version-aware derived catalogue index is implemented in `internal/query`.
- Library, book, and series HTTP endpoints delegate filtering, ordering, counts,
  and pagination to that transport-independent service.
- The common single-library query path retains only the requested page rather
  than allocating the complete result set.
- Series browsing is paginated and cover loading is explicitly viewport-bound;
  measured application and reverse-proxy baselines live in `PERFORMANCE.md`.
- The non-intrusive future media/shelf selection model is recorded in
  `MEDIA_NAVIGATION.md`; its switcher remains hidden until multiple real media
  families are configured.
- Acquisition/cache, settings, runtime health, reading/sync, and bounded
  background-job coordination are transport-independent services. Library
  scans, artwork extraction, podcast downloads, and HLS conversion all use
  process-owned worker pools with cancellation and observable queue state.
- The browser application is now split into cohesive feature modules for book
  cards and lazy artwork, collections, Continue Reading, metadata review, and
  settings. The remaining catalogue file coordinates shelf, series, title,
  search, and view state rather than owning those feature implementations.
- Persistent publication-file caching now belongs to `internal/acquisition`;
  HTTP delivery consumes its bounded cache API, while KoSync document hashing
  remains in the sync-facing web package.
- Managed configuration ownership now lives in `internal/settings`. It bounds
  and validates JSON documents, redacts credentials from reads, preserves an
  existing KoSync password when the UI submits a blank secret, and coordinates
  live apply or fallback save results. HTTP only maps requests and errors onto
  that service; configuration parsing and runtime reconciliation remain owned
  by `internal/appconfig` and the application layer respectively.
- Runtime reporting now belongs to `internal/health`. The public liveness
  endpoint stays constant-cost and intentionally minimal; the authenticated
  detailed endpoint returns a snapshot of shelf readiness and item counts,
  enabled optional services, and process uptime. It never performs a scan,
  contacts a provider, or exposes source and cache paths. Report evaluation and
  deterministic ordering are independent of HTTP and race tested.
- Reading/sync extraction has begun with the canonical position and text-anchor
  model in `internal/reading`. It owns the shared JSON contract, optimistic
conflict identity, and bounded validation policy for fractions, CFIs, device
IDs, and cross-renderer text anchors. Web and KOReader adapters use aliases to
preserve the existing on-disk format; durable store operations have since
moved into that package.

The reading store also defines the cross-renderer same-position invariant:
reflowable fractions within 0.2% are one effective location. Fixed-layout PDF
and comic positions require the exact page, because adjacent pages in a long
publication can fall inside that fractional tolerance. A stale write at an
equivalent location is acknowledged idempotently with the authoritative record
and does not advance history or replace its newer locator, timestamp, device
identity, or revision. Materially different stale positions still return a
conflict. Web and Android share this distinction, so renderer rounding and
lifecycle checkpoints do not create routine prompts or accept an adjacent page.

The durable reading store is now fully owned by `internal/reading`. Account
isolation, optimistic revisions, Continue Reading history, finished and
dismissed state, KoSync document links, accepted remote positions, and atomic
fsynced persistence no longer depend on HTTP types. The existing version-1 JSON
schema and state-file location are unchanged. Protocol adapters translate only
their wire records into the reading service model; direct service callers are
validated independently of HTTP. Persistence, reload, history, invalid-input,
and conflict behavior are tested in the owning package as well as through the
existing browser and KOReader regression suites.

KoSync coordination now also belongs to `internal/reading`. A dedicated
coordinator owns normalized document links, candidate ordering and
deduplication, compatibility match labels, remote-offer timing, validated
record/accept/seen transitions, and the bounded legacy-anchor projection sent
to devices. HTTP still owns authentication and wire decoding; catalogue-aware
filename and binary matching stays in web because it depends on configured
shelves. Production adapters no longer invoke sync-transition methods directly
on the store. Policy tests cover precedence, stale/seen candidates, legacy
anchor immutability, and invalid protocol state.
- Account-scoped collections now persist lightweight book and dynamic-series
  references and publish through both the web catalogue and OPDS. Catalogue
  fallbacks use natural numeric title ordering.
- The first removable movie slice now lives in `internal/video`: filesystem
  discovery publishes an immutable, naturally ordered snapshot with a
  persistent last-known-good cache. Movie libraries bypass publication, OPDS,
  metadata-review, reading-sync, and collection paths.
- `/api/videos` pages and searches only the selected movie shelf. Compatible
  files use a separately validated HTTP byte-range route and a native browser
  player; MKV remains indexed with an explicit unsupported-direct-play state.
- Account-scoped movie resume state is now a separate atomic store and the
  browser player saves during playback and on pause, completion, and close.
  Mobile fullscreen requests landscape orientation where supported.
- Poster extraction, television and season grouping, media probing, language
  preferences, playback resume, direct play, bounded optional HLS transcoding,
  and the initial Jellyfin adapter are implemented. Remote-storage video and
  trick-play previews remain deferred. The capability contracts are recorded
  in `DIRECT_VIDEO_CAPABILITY.md`.

## Delivery order

### 1. Stabilize the book platform

- Maintain a version-aware derived index for normalized search, buckets, stable
  ordering, and later facets. Rebuild it only after a backend publishes a new
  snapshot.
- Keep cold cover extraction ranged and concurrency bounded; keep resized cover
  and publication caches persistent, limited, and observable.
- Add benchmarks around large catalogue queries, series grouping, cache hits,
  and scan publication. Track cold remote work separately from warm requests.
- Continue race, malformed-input, pagination, and synchronization testing.

### 2. Separate application services

Move responsibilities out of the growing HTTP server without changing routes:

```text
HTTP / OPDS / web UI
        |
        +-- catalogue query service
        +-- acquisition and cache service
        +-- reading-state and sync service
        +-- settings and library-health service
        +-- bounded background-job service
                         |
                  storage capabilities
```

Suggested package boundaries are `library`, `query`, `acquisition`, `cache`,
`reading`, `sync`, and `jobs`. HTTP handlers translate requests and responses;
they do not implement indexing, storage, or synchronization rules.

#### Apply routine settings without restarting

Implemented as the first live-reconciliation boundary. **Save changes** writes
a temporary candidate, validates it, builds and scans a complete replacement
runtime while the current handler continues serving, atomically saves the
canonical configuration, and then swaps handlers. A validation or activation
failure leaves both the active runtime and last known-good configuration intact.
The rescan schedule is replaced without leaking the previous timer.

Routine library, cache, scan, and synchronization settings therefore take
effect without restarting the container. Genuine process-level settings, such
as the listen address or environment-managed authentication and secrets, still
require a restart and are reported as such by the settings API.

Diff-based reconciliation now retains unchanged library backends and their
published snapshots. Added libraries and changes to a source or its inference
rules scan before publication; presentation-only and process-wide changes reuse
the active catalogue immediately. The remaining refinement is to publish
per-library applying, scanning, ready, and error states if activation becomes
asynchronous in a future version.

### 3. Establish a media-neutral shell

Only after the book services are separated, introduce shared identity,
ownership, search, artwork, permissions, and playback-state concepts. Keep
book-specific fields such as EPUB locations and series indexes out of video or
audio models. Each media module owns its metadata and playback semantics while
using the common storage and job infrastructure.

The Library destination must also become a lightweight navigation boundary. It
must not implicitly query an `All shelves` catalogue spanning every configured
media family. Entering Library should restore the last selected family and
shelf when available, or show a cheap family/shelf chooser that performs no
catalogue query. `All shelves` remains an explicit choice scoped to the active
media family, so selecting Movies or Anime never first groups and sorts every
book. Counts and selectors may use compact precomputed index metadata; covers
and item rows load only after the user chooses a scope.

### 4. Finish the publication metadata workbench

Complete metadata inspection and repair as a bounded publication service before
introducing another media family. Preserve storage-specific write boundaries
and require review, verification, and rollback for every canonical change.

#### Metadata workbench extraction

Bring the useful behavior from `otherapps/metadatafixer` across as three small
services rather than embedding its monolithic server:

```text
metadata inspection (read-only, bounded workers)
        |
        v
durable review job (current / proposed / reason / confidence)
        |
        v
explicit storage apply capability (backup / verify / rollback)
```

The first release targeted EPUB files in explicitly writable ordinary-folder
libraries. Every scan is a dry run. Apply remains unavailable until the scan
finishes, uncertain authors or indexes are resolved, and the user confirms the
review. File renaming and category relocation are separate optional operations,
not side effects of metadata repair. Calibre libraries are read-only unless a
user explicitly enables the `calibredb` adapter. Seafile is permanently
read-only in Mochil: catalogue overrides may change how metadata is
presented, but the server never uploads, replaces, renames, or deletes
repository content.

Inference must use a per-library profile rather than assume one folder shape.
Profiles compose embedded metadata, parent-folder series, filename volume
markers, optional filename authors, and explicit overrides while recording the
source and confidence of every proposed field. Applying a job invalidates only
the affected catalogue entries and cover/publication cache keys.

The read-only workbench is implemented in `internal/metadata`. Settings can
inspect a selected publication shelf and review missing authors, missing
titles, missing series indexes, and duplicate series indexes. Each inspection
is stored as a bounded, versioned review job in `/data`, including conservative
filename/title-derived proposals with source and confidence. Inspection uses
the immutable published catalogue snapshot and performs no backend reads or
file writes. Saved jobs can now be reopened and filtered, and each finding can
be accepted with its proposed or a manually entered value, ignored, or left
pending. Decisions are atomically persisted and a fully decided job becomes
resolved. Apply remains guarded by a separate typed confirmation.

Resolved jobs now expose a storage-aware apply preview. Ordinary filesystem
EPUB changes are identified as eligible for the backup/validate/atomic-replace
transaction. Seafile previews use reversible catalogue overrides and never
offer source mutation. Calibre previews report whether its optional `calibredb` adapter is
available; Mochil never edits `metadata.db`, `metadata.opf`, or
Calibre-managed publication files directly.

The isolated local EPUB transaction engine is now implemented and tested. It
requires an explicit per-filesystem-shelf `metadata_writes` opt-in, rejects
symlinks and unsupported formats, creates an exclusive durable backup, rewrites
through an adjacent temporary file, validates the ZIP/container/package XML,
fsyncs the result, and atomically replaces the original. It is not yet exposed
as a generic storage operation. For explicitly opted-in ordinary filesystem
shelves, the complete apply endpoint is now available: it preflights the whole
resolved job, requires a typed per-job confirmation, serializes apply work,
rolls back already-updated EPUBs in reverse order after a failure, retains
durable backups, locks the job as applied, invalidates affected covers, and
rescans the shelf.

The Calibre adapter is now implemented behind the same per-shelf opt-in and
typed confirmation. It resolves database IDs by matching exact publication
paths from `calibredb list --for-machine`, captures each original record with
`show_metadata --as-opf`, applies only approved fields with `set_metadata`,
reads every record back for verification, refreshes Calibre's normal OPF
backups, and restores touched records in reverse order on any failure. Tests
use an injected command runner and never open a real Calibre library. The
normal container stays minimal; an optional Calibre-enabled container and
Debian/Incus instructions provide the CLI when needed. Seafile uses reversible
catalogue overrides only and remains permanently read-only.

Before media expansion, finish the high-value review workflow:

- propose series-level author repairs from consistent sibling publications;
- batch high-confidence title-derived volume corrections;
- distinguish duplicate files and editions from duplicate reading-order values;
- model side stories, prefaces, glossaries, omnibus editions, and intentional
  duplicate indexes without forcing a false numeric order;
- make deliberate ignores durable and reversible;
- keep uncertain and externally sourced corrections review-first.

The workbench now has a versioned, atomic catalogue-override layer for reviewed
corrections that cannot safely be written to their source. PDF, Seafile, and
write-disabled shelf corrections can therefore appear consistently in web and
OPDS while canonical files remain untouched. Mixed local apply plans identify
source-file updates and overrides separately. Missing-volume inspection now
reports only unambiguous title markers and excludes numeric ranges, removing
noise from complete editions, ordinary extras, and omnibus files.
Per-library preferred-format ordering now removes strongly matched cross-format
duplicates from published catalogue snapshots and metadata inspection. EPUB can
therefore rank above PDF without deleting either source. Matching requires an
exact normalized title plus a known author, or a series and volume identity;
weak matches and same-format editions are never collapsed. Underlying IDs stay
directly addressable, and an explicitly empty preference list disables the
filter.

### 5. Inventory Mochi by behavior

#### Mandatory pre-video code-health gate

No video playback implementation starts until a focused dependency and
cohesion audit is completed. The audit must verify package direction, storage
capability boundaries, persistent-state ownership, cancellation and resource
bounds, and regression coverage for books, OPDS, acquisition, reading, KoSync,
and metadata. It must identify and split files that own unrelated behavior;
line count is supporting evidence rather than the acceptance criterion.

The audit baseline identified two required extractions:

- `internal/web/ui.go` mixes static delivery, metadata workflows, catalogue
  queries, acquisition, covers, sync, and settings. Move metadata HTTP
  translation and apply orchestration behind a metadata service, then extract
  acquisition/cover delivery, before adding video routes.
- `internal/web/static/app.js` mixes catalogue browsing, collections, settings,
  metadata review, KoSync, and shell state. Split it into dependency-light
  screen/feature modules without introducing a framework or changing the
  existing visual system.

The gate is now complete. Browser behavior is divided into catalogue,
collections, reading dashboard, metadata review, settings, and reusable book
card modules. Server HTTP translation is divided into catalogue, metadata,
publication delivery, settings, and static-delivery files. Publication caching
belongs to acquisition, archive artwork interpretation belongs to artwork, and
catalogue filtering belongs to query. The video module must remain
independently removable and must not add conditionals throughout book code.

The first extraction is complete: `internal/metadata` now owns a deterministic,
storage-independent apply plan. Filesystem EPUB, Calibre, and catalogue-
override adapters consume the same grouped and validated field changes instead
of independently interpreting review decisions. EPUB and Calibre batch
transactions now also own deterministic preflight, durable backups, ordered
application, verification, and reverse rollback. Typed rollback failures retain
their operational severity at the HTTP boundary. The shared commit coordinator
now atomically combines the review-job state with optional catalogue overrides
and source rollback. The remaining web orchestration is limited to capability
selection, target construction, cover invalidation, rescan, and response.

The acquisition boundary has also started in `internal/acquisition`. Bounded
reads now resolve backend readers and temporary remote URLs through one
cancellable service with an explicit byte limit and injected HTTP client. Cover
and publication delivery now use the same service for seekable readers and
ranged remote responses. Archive cover interpretation moved to
`internal/artwork`, with its own compressed-container and decoded-size bounds.
Future media modules can depend on acquisition without importing web handlers.
Persistent publication cache ownership and HTTP publication/cover delivery are
now separately extracted and race tested.

Create one short capability record per Mochi subsystem. Record user-visible
behavior, inputs, outputs, dependencies, known failure cases, resource costs,
and what should deliberately be omitted. Do not copy Mochi's server structure,
global state, database schema, or UI shell.

Inventory direct video playback first, followed by television grouping,
audiobook playback, podcast refresh and downloads, music tags and queues, then
documents and wiki.
Inventory AI separately and deliver it last.

### 6. Build the media foundation without adding a player

Introduce only shared concepts proven necessary by publications and the first
Mochi inventory:

- stable library and media-item identity;
- capability-based storage access, including seekable and ranged streams;
- media-family registration and cheap per-family and shelf summaries;
- shared artwork cache keys and bounded thumbnail jobs;
- account-scoped collections, favorites, history, and playback state;
- durable bounded jobs with progress, cancellation, retry, and health;
- lazy APIs that return summaries before artwork or expensive metadata.

The shared job foundation now lives in `internal/jobs`. It uses a fixed worker
count, a bounded queue, keyed deduplication, cancellation, observable terminal
states, deterministic snapshots, and graceful shutdown. Podcast downloads are
the first migrated workload: they retain the existing two-worker limit,
six-hour timeout, per-episode size bound, and cache ceiling without spawning an
unbounded goroutine for every request. Deleting an episode cancels active work;
live configuration swaps close the replaced pool, failed candidate
configurations close their newly created pool, and process shutdown closes the
active handler. Transcoding and artwork generation will migrate separately
because their admission, restart, and caching semantics differ from downloads.
Stale podcast refresh is also server-owned background work now: feed-page
requests submit one deduplicated refresh key instead of spawning a goroutine per
request. It shares the handler lifecycle, queue bound, cancellation, and clean
configuration-swap behavior of the common job foundation.
Authenticated diagnostics can inspect both the server queue and podcast
download queue through the read-only `/api/jobs` endpoint. The response exposes
only queue name, stable job key, lifecycle timestamps, and status; it does not
expose source paths, raw worker error text, or mutable controls.
Audio-cover and video-poster extraction now use the same bounded job primitive
with separate two-worker pools and 64-item admission queues. Concurrent requests
for the same content key coalesce, shutdown cancels FFmpeg through context, and
configuration swaps close both generators. This also removes the old artwork
key-lock map, which retained one mutex for every image ever requested.
Compatibility HLS admission now also uses the common pool with two workers and
a 32-job queue. Seek restarts cancel the superseded keyed job while preserving
the existing stable segment cache and near-requested-position restart behavior.
Transcode state appears in `/api/jobs`, idle sessions remain cancelable, and
handler replacement or shutdown now terminates active FFmpeg processes. Generic
terminal job history is capped as well, so unique downloads, artwork keys, and
seek generations cannot grow operational state without bound.
Trick-play frames now use a separate two-worker queue and bounded persistent
cache, so timeline hovering cannot delay posters or HLS work. Requests are
quantized to stable ten-second keys, revision-aware, generated only on demand,
and omitted on touch layouts where native seeking remains clearer.
Both podcast browse routes now call one refresh scheduler. The audio-family API
no longer bypasses job admission with a request-created goroutine, so opening a
show and opening the podcast overview coalesce behind the same stale-refresh
key.
The delayed server-restart action is keyed and admitted through the same server
pool too. Repeated restart requests coalesce, queue failure is reported instead
of silently spawning work, and shutdown can cancel the delay cleanly.
Cover-cache migration and publication-cache pruning now run as deduplicated,
cancelable server jobs rather than detached startup goroutines. A completed
publication download schedules pruning after the response-path commit instead
of walking the cache synchronously, while the same byte and expiry ceilings
remain enforced.
Seafile author hydration remains a single-flight store task, but now inherits a
store-owned cancelable context. Rejected configurations close newly constructed
stores, changed or removed libraries close the replaced store after the atomic
handler swap, and shutdown cancels scheduled rescans before closing active
stores. A canceled hydration waits for its HTTP response resources to unwind.
Initial and live-added audio scans now use a process-owned two-worker queue
instead of one raw goroutine per shelf. Startup remains non-blocking for audio
metadata probing, while admission is bounded, duplicate fingerprint scans
coalesce, scan state appears in `/api/jobs`, and shutdown waits for active scan
workers before closing their libraries.
Scan job keys expose the shelf ID plus a short fingerprint digest only; the
diagnostics API never returns the underlying configuration fingerprint or its
mounted source paths.
Periodic rescans now submit every shelf through that same process queue instead
of scanning serially inside the timer goroutine. An already-running scan for an
unchanged shelf is deduplicated, at most two shelves scan concurrently, and a
scheduler reset waits for the prior timer loop to stop before replaced stores
are released.
Reload also cancels and drains any already-admitted scan for a changed or
removed shelf before closing that backend. Unchanged fingerprints retain their
store and active scan; changed fingerprints receive a distinct queue key.
Detailed health now includes aggregate lifecycle counts per queue. Monitoring
can see queued, running, ready, failed, and canceled work without receiving job
keys, worker errors, media paths, or source configuration.
Filesystem book, video, and Seafile stores now serialize `Scan` internally,
matching the existing audio and document contracts. This protects callers that
do not enter through the runtime queue, including metadata-apply rescans.
Seafile cache writes also serialize separately so author hydration cannot race
a later catalogue scan while atomically replacing the same cache file.

Opening Library must never enumerate every media family. Restore the last
family and shelf or show a cheap chooser before querying catalogue items.

### 7. Add video and television

Begin with direct play for browser-compatible files and HTTP range requests.
Keep movie metadata and episodic grouping in the video module. Add season and
episode inference, artwork, resume state, and collections after direct play is
reliable. Transcoding is a later optional capability with strict worker,
cancellation, and disk limits; it must never be required by the book server.

Compatibility playback now uses a complete synthetic VOD timeline with stable
two-second segment numbers. Missing regions restart the bounded FFmpeg worker
near the requested segment while retaining already generated segments in the
LRU cache. This gives HEVC and otherwise incompatible files an honest duration
and seekable timeline without pre-transcoding an entire film. Direct-play media
continues to bypass this path. Admission status is exposed through the
authenticated job API, failed sessions use bounded exponential retry cooldown,
and height plus cache ceilings are configurable. Trick-play preview generation
remains a separate future bounded job.

Artwork follows one shared artifact contract rather than provider-specific UI
fields. Resolution order is: reviewed local sidecar, embedded artwork, locally
generated video frame, then an explicitly enabled metadata provider. Originals
and display thumbnails have content-addressed keys, separate size limits, and
bounded generation workers. Movies use posters and optional backdrops; shows
and seasons may override the show poster; podcast feeds and audio tags supply
their own artwork through the same cache interface. Catalogue APIs return URLs
and dimensions only, never image blobs or provider behavior. Provider lookup
remains optional, reviewable, and outside the scan critical path.

### 8. Add podcasts independently

Implement feed discovery, conditional refresh, stable episode identity,
streaming, resume position, played state, and an explicit bounded download
queue. Feed failures retain previously published episodes. Retention and
automatic downloads are opt-in policies with visible byte limits. Podcast
state does not inherit music queue behavior merely because both contain audio.

Implemented foundation: RSS discovery, stable GUID/enclosure identities,
conditional background and manual refresh, last-good retention, same-origin
range streaming, durable resume, and a two-worker bounded download cache.
Played state now reuses the durable podcast progress record, including automatic
completion at 98%, explicit mark-played/unplayed actions, and server-side
All/Unplayed/Played filtering before pagination. Each feed can independently
opt into automatic downloads and retain 1–50 newest episodes. Reconciliation
runs after subscribe, manual/background refresh, and policy changes through the
two-worker downloader; subscription removal also removes its cached episodes.
The management UI reports actual use against the global bounded cache ceiling.
The podcast overview is built from the complete subscription index rather than
paginated episodes; an individual show's episodes are fetched only when it is
opened. This keeps every subscribed show visible without loading the full
episode catalogue.
Subscription management now surfaces persisted per-feed refresh health. Healthy
feeds show their last check, failed feeds keep publishing the last-good episodes
with a concise error, and the existing manual refresh becomes an explicit local
retry/loading state instead of an unhandled request.
Failed automatic downloads are included as an aggregate in the management
surface and can be explicitly re-admitted through the existing bounded queue.
The API returns only the count and retry result; raw worker errors remain out
of the general management UI and operational diagnostics.

Frontend ownership is split as well: the shared audio controller owns playback
and durable state coordination, pure family view helpers own podcast/music
presentation, and the podcast manager owns discovery, subscriptions, refresh,
download policy, and retention UI. Audiobook chapter rendering, highlighting,
drawer controls, and direct seeking live in a separate chapter component. Music
queue, favorites, and playlist persistence live behind a dedicated state module.
ReplayGain calculation and Web Audio graph ownership live in a separate player
component rather than expanding the shared controller.
The music queue is directly browseable and editable. Rapid browser mutations
are serialized and coalesced, durable stores use rollback-safe atomic updates,
and bounded startup validation rejects oversized or malformed queue and
playlist state.

### 9. Add audiobooks independently

Read embedded chapter, author, narrator, series, duration, and artwork metadata
without treating audio files as publication files. Add chapter-aware playback,
resume state, sleep timer, playback speed, and durable progress sync. Reuse the
shared media identity and acquisition boundaries, but keep audiobook progress
and navigation separate from podcast played-state and music queues.

Implemented foundation: bounded parallel tag/chapter probing, embedded and
sidecar artwork, direct range streaming and downloads, durable resume,
Continue Listening, speed controls, chapter context and direct chapter seeking,
natural series/volume ordering, narrator display, and a sleep timer. When tags
are incomplete, the explicit `Author/Series/Book NN/file` layout safely supplies
author, series, and volume without affecting normal music folders; AppleDouble
resource-fork files are ignored.
Unsupported browser decoders fall back to bounded, seekable AAC/fMP4 HLS while
the original source remains available for direct playback and downloads. The
codec-neutral segmented engine owns source identity, process serialization,
validated file descriptors, cache eviction, and manifest addressing.

### 10. Add music independently

Read embedded tags and artwork with bounded workers. Build artist, album, disc,
and track indexes with deterministic natural ordering. Add direct playback,
persistent queues, and favorites. Audio analysis remains optional rather than
a prerequisite and runs only after an explicit per-shelf request.

Implemented foundation: bounded tag/artwork probing, deterministic artist,
album, disc, track, and title ordering, direct range playback/downloads, and a
durable account queue API. Track, artist, album, favorites, and playlist browse
surfaces are account-aware and avoid full-library loading. Standard track and
album ReplayGain tags (plus R128 fallbacks) now drive a bounded, peak-aware Web
Audio gain stage with a persisted Track/Album/Off control. A dedicated
dual-deck transport preloads only the next queued track (up to 128 MiB), swaps
decks when ready, and otherwise preserves the ordinary transition path. Audio
Missing loudness values are measured by a bounded EBU R128 analyzer that
preserves embedded tags, revision-checks results, atomically persists each
batch, and never rewrites source audio.

### 11. Add documents and wiki after media stabilizes

Define supported formats, indexing, permissions, revisions, and search before
importing Mochi behavior. User-authored documents require an explicit backup
and revision model and cannot live only in a disposable catalogue cache.

Documents and wiki are two modules, not two views over the same storage:

- **Document libraries are read-only source indexes.** The implemented formats
  include Markdown, plain text, PDF, and DOCX on ordinary filesystem shelves. A scan reads
  directory metadata only, skips symlinks and resource forks, publishes an
  immutable last-known-good snapshot, and never rewrites source files. Stable
  identity is library plus relative path; revision is path, size, and modified
  time. Additional formats and remote backends require explicit capabilities.
- **Document content and search are bounded derived work.** Text, Markdown, and
  DOCX reads plus previews and indexing have per-file byte/page/text limits,
  cancellable worker queues, and versioned persistent caches. Unsupported or
  oversized files remain downloadable instead of being partially interpreted.
  Search is library-scoped and paginated; merely opening Library does not
  extract every document.
- **Wiki pages are account-scoped authored state.** They live in a separate,
  versioned data store under the persistent data directory. Every write uses an
  expected revision, creates an immutable prior version, and commits by atomic
  replace. Delete is recoverable until explicit purge. Bounded history,
  restore, full export, and backup are prerequisites rather than follow-ups.
- **Markdown is untrusted input.** Rendering escapes raw HTML by default,
  permits only an explicit safe element/attribute and URL allowlist, and never
  executes scripts, event attributes, remote embeds, or unsafe URL schemes.
  `[[wiki links]]` and backlinks are derived indexes, not embedded authority.
- **Permissions stay explicit.** Imported documents inherit shelf read access;
  wiki pages belong to an authenticated account. Sharing and collaborative
  editing are deferred until authorization, conflict UI, audit events, and
  retention rules are specified.

Delivery slices are: (1) isolated read-only document scanner and cache,
(2) paginated document API and minimal library surface, (3) bounded on-demand
text extraction and search, (4) versioned wiki store and API, then (5) editor,
links, backlinks, history, restore, and export. Mochi's useful behavior informs
these contracts; its AI features remain part of phase 12.

Implemented foundation: the isolated scanner indexes Markdown, plain text,
PDF, and DOCX files from filesystem shelves without modifying their content.
It skips symlinks and resource forks, publishes an immutable cached snapshot,
and retains the last known-good catalogue when a source is unavailable. The
runtime now recognizes `documents` as a separate backend, preserves unchanged
snapshots during live configuration reconciliation, and exposes a bounded,
paginated, library-scoped API plus secure inline/download delivery. The
settings option remains hidden until the document library surface is complete.

The document surface and bounded text-search slice are now implemented. Users
can browse in remembered list or cover layouts, open PDFs natively, and read
UTF-8 Markdown, text, or bounded DOCX content in a safe in-app reader. DOCX
extraction reads only `word/document.xml`, caps both the archive and expanded
XML, and renders escaped text. Raw HTML is escaped. Reader output is limited to
2 MiB per file; the persistent revision-aware search index uses
four workers and caps indexed content at 64 MiB per shelf. Search returns
context snippets and explicitly reports partial coverage when oversized or
invalid files cannot be indexed. Source folders remain read-only.

The versioned wiki store and API foundation are now implemented separately
from document shelves. Pages are isolated by authenticated account, require an
expected revision for every update, deletion, restore, and historical restore,
and retain up to 50 immutable prior versions. Deleted pages remain in a
recoverable trash view. Writes use a synchronized temporary file, fsync, and
atomic replace under the persistent data directory. Page content is valid
UTF-8 and limited to 2 MiB, account stores are limited to 10,000 pages, and a
complete JSON export includes both current pages and history. The editor,
wiki-link index, backlinks, history UI, and guarded restore are implemented.
Permanent deletion requires seven complete days in Trash, the current page
revision, and exact typed confirmation; the page and its history are then
removed atomically.

The first wiki workspace is now implemented. It provides a responsive page
browser, Markdown editor, safe rendered reading view, `[[page links]]`, derived
backlinks, revision history and guarded restore controls, recoverable trash,
and export from the persistent account store. History can compare bounded line
changes against the current page before restoring; very large notes fall back
to the complete earlier snapshot without blocking the browser. Missing wiki
links can create a new titled page. Raw HTML remains escaped by the renderer,
conflicts are shown without overwriting the editor, and destructive or discard
actions ask first. Safeguarded permanent purge is implemented. Collaborative
sharing and richer Markdown remain deliberately deferred.

### 12. Add integrated AI last

AI is an optional service boundary, not a catalogue dependency. Indexing and
playback must work when it is disabled or unreachable. Providers, model data,
quotas, background work, privacy controls, and generated metadata provenance
must be explicit.

AI memories and annotations use an account-scoped, exportable, deletable store.
Generated values never silently replace source metadata, and media content does
not leave the server without explicit provider and privacy configuration.

## Scale checkpoints

- At tens of thousands of items, use precomputed per-library sorted and facet
  indexes and avoid rebuilding result sets for offset pagination.
- Video artwork, scans, and transcoding use bounded queues and health reporting;
  keep those limits explicit as additional media providers are added.
- Transcoding prefers direct play, has worker admission and cancellation, a
  configurable bounded cache, and exponential retry cooldown after failures.
- Before multiple users, separate authentication identities, permissions,
  reading/playback state, cache visibility, and administrator settings.
- Before enabling several large media families, ensure merely entering Library
  does not enumerate, group, sort, or fetch artwork for every shelf.

## Definition of a safe extension

A new feature is ready when disabling it restores existing behavior, failure is
isolated to its module or library, persistent changes are migratable, expensive
work is bounded, and the complete existing regression and race suites pass.
