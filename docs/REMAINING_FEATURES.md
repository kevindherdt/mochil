# Remaining features

For the latest inspected split between published work, unpublished fixes and
confirmed gaps, see [Current delivery status](CURRENT_DELIVERY_STATUS_2026-09-08.md).
The detailed history below includes completed foundations as context.

> Architectural constraint: new work must not create godfiles or broad monoliths. See [MODULARITY_AUDIT.md](MODULARITY_AUDIT.md) for enforceable size guidance, grandfathered hotspots, and the extraction order.

The ingest, naming, matching, editable-source, server-migration, and
library-specific metadata decisions are specified in
[`INGEST_AND_LIBRARY_MANAGEMENT.md`](INGEST_AND_LIBRARY_MANAGEMENT.md).

This document summarizes the work that remains after the currently implemented
Mochil stabilization and media foundations. For full architectural context,
constraints, and implementation history, see `ARCHITECTURE_ROADMAP.md`.

## Android client

The detailed mobile-first delivery plan is maintained in
[`ANDROID_ROADMAP.md`](ANDROID_ROADMAP.md). It defines one consistent grouped
browsing model for books, music, audiobooks, podcasts, and later television,
followed by separate Android TV and Kobo compatibility tracks.

The native Android foundation now lives under `android/`. Its first real server
contract accepts the ordinary Mochil HTTPS address, username, and password,
then exchanges those credentials for a bounded, revocable per-device token;
only the token hash is persisted by the Go server, and the client contract keeps
the returned token in Android Keystore-backed encrypted storage. Catalogue,
content, and progress are separate client interfaces so playback, downloads,
and the future native reader do not grow into one controller.

The executable extension experiment has been removed from every client and the
server for store readiness. Generic OPDS, WebDAV, RSS, local media, and
self-hosted Mochil connections remain supported. The removed executable-code
path is not part of the roadmap and must not return to either distribution.

Authenticated library discovery, cursor-paginated book browsing, automatic
scroll continuation, and private device-token cover loading are now implemented.
Client navigation now detects Android TV and dedicated e-ink devices. TV hides
reading shelves by default, e-ink hides video shelves by default, and a dedicated
Settings section can show or hide every server library per device without
changing the server catalogue.
Publication shelves also expose a server-sorted Recently Added view on mobile
and TV. Its cursor is bound to the recent view and its warm deep-page path uses
the versioned publication index rather than sorting the full shelf per request.
Publication Authors browsing is also server-owned and cursor-paginated on
mobile and TV. It supports cover/list roots, search, alphabet filtering, and
author detail without grouping only the currently loaded client page.
Author and series details retain independent bounded continuation cursors on
mobile and TV; root shelf cursors never leak into an open group.
Publication and music browse modes use the same compact checked menu on mobile,
leaving more room for covers and titles without hiding the active choice from
accessibility services.
Media3 background music playback, system media controls, secure native streaming,
the system-inset-safe persistent mini player, the complete server-backed
Artist → Album → Track hierarchy, cover/list views, and the adaptive launcher
identity are now implemented. Publication shelves now provide Series and All
views, independent cover/list layouts, lazy series volumes, series and book
details, search, downloaded filtering, cursor continuation, and alphabet jumps.
App-private downloads plus native EPUB and CBZ reading are active. Interrupted
publication and audio transfers now resume only from validator-backed partials:
Range/If-Range continuation requires an exact Content-Range match, while ignored
or stale ranges restart without mixing file versions. Startup removes malformed
and orphaned partial state but retains recoverable work. Foreground WorkManager
queue controls remain a later explicit-download UX slice; immediate publication
opening remains direct. Audiobook
playback now includes chapters, device-local position bookmarks, native speed
controls, and a service-owned persistent sleep timer that remains active when
the player sheet is dismissed; chapter metadata remains available for offline
downloads and restored Media3 sessions. The service also restores a bounded
active queue, item, position, speed, repeat, and shuffle state after process
recreation without persisting credentials or a hard-coded server address. EPUB
highlights and notes synchronize through the account annotation store in both
directions. KOReader-origin highlights carry stable IDs, native locators, text
anchors, revisions, and deletion tombstones into Android; Android-origin EPUB
highlights resolve their text anchors into native KOReader annotations while
preserving one cross-device identity and guarding transient scans against false
deletion. One versioned interoperability corpus now exercises Android and Go
decoding against the same Readium locator, KOReader XPointer, text-anchor, and
stable-identity payloads so either side cannot silently drift. Settings exposes bounded export and exact-confirmation deletion of
the server copy. The native Highlights tab can synchronize one publication and
explicitly create a bounded Mochil Notes page without adding permanent reader
chrome. Reader navigation separates Contents, Bookmarks, and
Highlights, and saved decorations are reapplied when each Readium resource is
ready. The EPUB reader defaults to immersive full screen, can restore Android's
time/network/battery status strip on request, and independently keeps progress,
estimated chapter time, and estimated book time visible in its optional
E-ink-safe footer. Native podcast browsing now provides Shows → Episodes, cover
and list layouts, unobtrusive search, played filtering, cursor continuation,
resume-aware playback, and app-private episode downloads. Episode ordering is
server-paged and device-persistent across newest, oldest, and natural title
order. Played state can be changed from each episode and reversed from the
Played filter without losing the episode. Podcast feeds can be
discovered through bounded directory search or added by RSS URL, refreshed on
demand, removed with confirmation, and assigned bounded shared server-cache
download and cleanup policies from the native Shows screen. Per-show device-local
auto-download, retention, and unmetered-network policies are also implemented;
downloaded episodes remain browsable while the server is unavailable. Existing files in a
server-local podcast library are linked to RSS episodes only when normalized show and
episode titles identify one unique file. Linked episodes retain their RSS progress identity,
stream the existing local file without rewriting it, and appear in the Downloaded filter.
Privacy-safe diagnostics distinguish device matches, server-local matches, unmatched files,
and an empty local index without uploading titles or paths. Podcast playback now
shows item-local feedback while refreshing the latest account position. Only the
newest explicit play request may start; leaving the podcast surface, changing
shows, or selecting a different episode invalidates an older in-flight lookup,
so a slow response cannot unexpectedly start stale media. A provider failure
falls back to the last known position rather than leaving the tap inert, with
an explanatory status on the mobile podcast surface. The optional refresh has
a short playback-specific connection/read deadline; it cannot inherit the
longer catalogue timeout and hold Play indefinitely on a degraded network.
Mobile video now
provides Movies and Shows → Seasons → Episodes, cover/list layouts, compact search,
resume-aware Continue Watching, direct playback, and server HLS fallback. Persistent
Mochil, E-ink, Manjaro-inspired, Ubuntu-inspired, and Material-inspired themes can
follow the device or use explicit light/dark brightness. Video picture-in-picture and
in-player audio-track selection are implemented. Playback failures distinguish unsupported
device codecs, malformed media, network loss, server delivery, and missing local storage with
concise recovery guidance and privacy-safe diagnostic categories. Release hardening now includes signed
release enforcement, expired-session recovery, strict backup exclusions, low-storage and
offline-startup handling, bounded process-recovery playback snapshots, and explicit
TalkBack actions for media, publication, queue, download, Home, mini-player, and reader
tap targets. Hardware accessibility and interruption verification remains alongside Android
TV refinement. The explicitly approved TV groundwork
now makes the shared APK visible to Leanback launchers and selects dedicated D-pad-safe
ten-foot Home and library surfaces. Bounded continuation rows, current-audio control,
virtualized cover grids, library switching, series/show/group drill-down, and remote
launching cover publications, music, audiobooks, podcasts, and video. A dedicated
Settings surface exposes update checks/install and connection management. TV library
routes subscribe only to their visible media state so playback clocks and unrelated
mobile features cannot invalidate a cover grid. Home confines playback-clock updates
to its compact transport. TV book, music, audiobook, and podcast grids continue
automatically near the end without a focusable Load more tile; the shared paging
policy retains virtualization, prevents duplicate requests, and exposes one quiet
loading tile. The self-hosted download now uses a minified, resource-shrunk
non-debug runtime while retaining the installed development certificate, and TV Settings
can opt into and send the same bounded timing/crash diagnostics as mobile. Video controls
now start hidden on TV and appear transiently on remote input. Active-shelf TV search
stays icon-only until invoked and submits explicitly from the remote keyboard. Rich
details, final ten-foot visual refinement, and the remaining device preferences stay in
the TV presentation track. A compact global transport now keeps audio and podcast
playback visible away from Home without invalidating the underlying library grid.
The full TV player keeps queue management progressively disclosed behind one
Queue action. Its virtualized rows mark the current item and support play,
move-earlier/later, and remove through D-pad Select and hold; ordinary player
chrome stays unchanged when the queue is closed. Starting TV music and editing
music or audiobook queues update both Media3 and the account-persistent order,
while podcast queues remain local to the current listening session.
Opening a long queue initializes its virtual viewport at the bounded current
item and restores focus there once. Queue mutations retain ordinary focus
ownership instead of repeatedly pulling the remote back to the playing row.
Closing Queue now explicitly returns focus to the main transport, and visiting
Now Playing preserves the prior library grid scroll and focused item so Back
returns to the same browsing context.
Shoulder navigation now passes through when there is no adjacent visible
library, avoiding a same-shelf reload and unnecessary focus reset on single-
library TV installations.
TV root shelves now provide remote-native browse modes for publication series or
all titles, music albums/artists/tracks, and audiobook series/authors/titles;
the compact selector disappears during drill-down to preserve the ten-foot hierarchy.
Focused covers now fill a stable-height TV detail area with bounded title,
metadata, resume timing, and descriptions where available. Independent native
focus-restorer groups keep the last D-pad target in the library strip, browse
selector, and virtualized catalogue when focus returns.
TV navigation now treats Home as the library switcher and each opened library as
a focused workspace, so unrelated library cards do not consume the catalogue view.
Back first closes the active show, season, artist, album, series, or author context,
then returns to Home with the previously selected library focused. L1/R1 remains an
optional shortcut for experienced remote or controller users to switch libraries
without reopening the switcher. Root browse modes such as Discover, Recent, Albums,
and Artists share one compact remote-friendly picker in the header instead of consuming
a second rail; focused-card context shows only useful metadata rather than repeating
the title already visible on the card. Home also avoids a persistent product-name
banner: Settings lives alongside the Libraries heading so the first row is actionable
and more catalogue content remains visible. The same content-first shell applies on
Android mobile and web: the persistent wordmark row is removed. Mobile Home and opened
libraries now share one virtualized, snapping destination rail for Home, server libraries,
Saved, Sources, and Settings instead of competing menus or permanent refresh/settings
buttons. Home activity refreshes automatically. Continue Reading, Listening, and Watching
rows keep their content width and expose the persisted Remove from Home action through a
long press and the equivalent TalkBack custom action rather than another visible icon.
TV and anime episode playback now uses a bounded, validated Media3 queue for the
current season. Its next controls remain part of transient player chrome;
automatic transitions complete the old episode, manual skips retain its exact
position, and ordinary movies remain single-item sessions.
TV sign-in now uses a ten-minute pairing code instead of remote-entered account
credentials. An authenticated browser explicitly approves the visible code;
an unguessable polling secret can exchange it exactly once for the existing
bounded, revocable Android device token. The account password never reaches or
is stored on the television.
TV Settings also exposes a compact, opt-in hardware readiness check. It records
D-pad, Select, and media keys without consuming normal focus navigation, reports
the Android app-memory class and HDMI-output presence, and makes the intended
display policy explicit. Video keeps the display awake; background audio does
not prevent the television's screen saver. The remaining release evidence is
physical-device coverage rather than another hidden application contract.

Video catalogue cards on mobile and TV now expose exact item watch progress with a thin
artwork-edge line and a completed state while leaving unstarted media clean. The TV
alphabet navigator uses one horizontal server-backed rail, music supports persistent
compact cover/list layouts, and Favorites starts the actual visible queue. Mobile and TV
share device-local subtitle presets for clean shadowed, cinema, outlined animation, and
high-contrast presentation, with bounded size/background controls and live preview;
Media3 applies the chosen caption style without accepting embedded font-size overrides.
While browsing an unfiltered TV catalogue, the single-line alphabet rail now follows
the currently focused card; an explicitly selected jump bucket remains authoritative.
This provides position feedback without adding a second heading or persistent control.
The video surface keeps Media3 transport transient and exposes advanced audio-track,
subtitle-track, rotation, picture-in-picture, and close actions through one compact More
menu that disappears with the controller instead of placing permanent icons over video.
Web video progress now distinguishes automatic direct/HLS restoration from a
deliberate browser scrub. Only the latter may override newer cross-device
progress; automatic reconnect and track-switch seeks remain passive. Closing
the web player commits one final checkpoint, relinquishes item ownership, and
then pauses, preventing duplicate close-time revisions and stale callbacks.
Android audio applies the same ownership boundary to server-derived podcast and
audiobook resumes. A one-shot token identifies only the matching automatic
Media3 seek, while subsequent user scrubs remain explicit.

Standalone sources now include encrypted OPDS 1.2 connections from local Home
and connected-server Settings. Nested Atom shelves, pagination, direct and
descriptor-based OpenSearch, same-origin Basic authentication, and bounded
EPUB/PDF/CBZ acquisition feed the existing offline index and native readers.
OPDS 2 JSON browsing now shares the OPDS 1 connection and source contract,
including publications, groups, navigation, pagination, templated search,
artwork, and acquisitions. Authenticated OPDS artwork loads
only for visible rows through an exact-origin, redirect-validated, concurrency-
and size-bounded local cache. A
unified connected-mode Sources destination now exposes OPDS and WebDAV through
the existing Home/library selector without expanding the bottom dock. Read-only WebDAV connections now
provide encrypted credentials, bounded folder browsing, supported-media
filtering, ETag-keyed private downloads, and native publication, audio, and video
opening. A bounded whole-tree scanner now reuses folder validators, keeps its
disposable index encrypted, reports restrained progress, and supplies global
filename search without changing the remote source. Connection changes cancel
stale folder and index work. Audio and video now probe range behavior before
opening. Compatible servers stream through an opaque, process-restorable Media3
reference whose credentials remain encrypted and exact-origin; incompatible
servers retain the bounded private-download path.
Recently browsed folders also retain a bounded encrypted last-known-good
snapshot and use provider folder ETags for conditional refresh where available.
Apache-compatible name-only XML document declarations are stripped before the
hardened folder parser runs; declarations with external identifiers or entity
subsets remain rejected.

An account-scoped external-library contract separates logical membership from
physical downloads for declarative OPDS connections. Android retains actionable
locations and credentials only in encrypted device storage, keeps an encrypted
offline cache, and routes externally acquired EPUB progress through the same
revision-safe reading contract. The server exposes only account-authorized Saved
membership and progress; it does not install, execute, proxy, or preserve
third-party extension runtimes. Web Settings retain the administrator-controlled
download root and quota, while Android SAF folders remain device-only.

Android EPUB restoration now checkpoints the exact Readium page immediately
and synchronously when backgrounded, independently of the structural locator
used for KOReader matching. Offline positions remain durably pending and retry
on reopen without requiring another page turn. Transient reader controls expose
a compact cloud state and the last server acknowledgement time; a failed or
offline write remains clearly identified as safely saved on the device.
Native EPUB and PDF writes, including device-read OPDS EPUBs, now identify the actual Android manufacturer/model
instead of labeling every client identically. A divergent-position prompt names
the device that saved newer progress and appears only when this device also has
unsynchronized progress; equivalent positions are acknowledged without moving
the exact local page. Device labels are bounded independently at the server
boundary.
KOReader-origin progress now resolves its bounded CREngine DocFragment path to
the matching Readium spine and CSS element before text-anchor or percentage
fallback. Android performs the inverse DOM-element conversion only for a
validated first-visible selector and stores that XPointer with the account
position, allowing the paired BoboSync plugin to consume a native KOReader
location without exposing publication content. Character-offset conversion for
arbitrary EPUB DOM normalization remains fixture-gated.

## 1. Metadata workbench refinements

The review workflow, durable decisions, catalogue overrides, local EPUB
transactions, optional Calibre adapter, and per-library preferred-format
selection are already implemented. Strongly matched cross-format publications
publish only the highest-ranked format while every source remains directly
addressable; weak matches and same-format editions remain visible. Grouping is
edition-aware: conflicting known languages, publication identifiers, reading
order positions, and special-publication roles remain separate, while one
format with sparse metadata can still match a single unambiguous edition. Explicit
volume and book markers in titles can now be accepted as a bounded batch, while
ambiguous part markers and ranges remain in individual review.
Side stories, prefaces, glossaries, omnibus editions, and companion material
are classified separately from the main reading order. Filesystem numbering is
not assigned to those roles, and metadata inspection neither reports their
intentional index collisions nor proposes a missing numeric index for them.
Seafile is an intentionally read-only source. Metadata review can apply
reversible Mochil catalogue overrides, but Mochil never uploads,
replaces, renames, or deletes Seafile files.
Apply-preview capability selection and deterministic filesystem/Calibre target
construction now live in the metadata package. Shared post-commit effects own
cover invalidation and rescanning, leaving HTTP handlers responsible for request
validation and response mapping rather than storage-policy decisions.

A storage-independent ingest inference package now describes every proposed
field with its source and confidence instead of returning an unexplained parsed
name. It conservatively recognizes explicit publication volume and ISBN markers,
Author/Series publication folders, movie years and provider IDs, television
SxxEyy plus Season folders, and music Artist/Album/disc-track layouts. Bare years
and unrelated numbers are deliberately not treated as ordering metadata. The
publication path is already connected to Metadata review: missing-author and
missing-title corrections derived from structured folders and filenames appear
as reviewable medium-confidence proposals. Embedded and sidecar metadata remain
higher-priority scanner evidence. Candidate matching now uses deterministic,
explainable score contributions: exact strong identifiers dominate, conflicting
values for the same identifier reject a match, and title-only similarity remains
low confidence. Stable candidate IDs break equal-score ties so provider order
cannot silently change a result. Further media-specific workbench screens and
batch-level conflict review remain to be built.

The first ingest planner is now available from the selected publication shelf
in Metadata settings. It is preview-only and fill-only: local filesystem and
Calibre catalogues can show missing title, author, series, series index, and
identifier proposals, but the endpoint accepts no mutation method and refuses
remote sources. Existing non-empty embedded metadata is never proposed for
replacement. Results are path-stable, limited to 100 items per response, and
continue automatically in the browser. The server builds the immutable plan
once per catalogue generation, so later pages do not repeatedly copy, infer,
and sort a large catalogue. The versioned managed-ingest journal and its
deterministic recovery planner are now tested independently: paths are canonical
and relative, state transitions are monotonic, persistence is bounded and
atomic, and an ambiguous target stops for manual review instead of being
overwritten. The first internal executor is now implemented for dedicated POSIX
incoming and managed roots. Planning creates no media file; execution revalidates
confined regular sources, enforces a per-file size bound, copies into private
temporary files, records SHA-256 verification, and commits with an atomic
no-clobber link. Sources are never moved or deleted. A target that appears after
planning is preserved even when its bytes happen to match, and partial failures
remove only targets proven to have been created by that execution attempt.
Completed journal history rotates without evicting active or recoverable
transactions. The executor now applies the recovery planner for provably safe
states: interrupted copies restart from clean temporary files, and a committing
transaction is finalized only when every target matches its journaled size and
digest. Mismatched or partially committed multi-file state remains untouched for
manual review. The live configuration schema and web Storage settings now
provide a collapsed opt-in boundary for the two non-overlapping roots and the
per-file limit. Relative paths resolve against the managed config directory;
invalid, incomplete, overlapping, or unbounded configurations are rejected
before a new live handler is published. A configured executor is constructed
with the process and retains its journal under the cache directory. The
administrator-only API now separates journal planning from execution, requires
the exact transaction ID before execution or recovery, and dispatches cancellable
copies through the bounded job pool. Public job diagnostics receive only a safe
summary; detailed filesystem state remains in the administrator-only transaction.
The Metadata workspace now exposes a collapsed two-stage browser workflow:
reviewing a relative source and destination creates an immutable plan without
copying, exact transaction-ID confirmation starts the bounded job, and cancelling
an unstarted plan durably closes it without touching media. The browser polls the
administrator-only transaction state and keeps error details adjacent to the
workflow. Runtime root resolution also rejects symlink-resolved nesting, so the
incoming and managed trees cannot alias or overlap.
Incoming discovery is now connected to that workflow as a bounded read-only
inventory. It recognizes an explicit publication/audio/video extension
allowlist, never follows symlinks, excludes oversized files, supports stable
cursor pages and kind filters, and stops after a fixed traversal budget. Each
row carries the conservative evidence proposal plus an editable deterministic
target path; selecting it only fills the review form and cannot create a journal
entry or media file. Episodic inference also removes a repeated show-name prefix
from filenames such as `Show.S02E03.Title.mkv`, preventing duplicated target
names while retaining the original source path.
Incoming discovery now also supports a bounded selection of up to 100 files for
one read-only batch review. The review reruns source confinement, regular-file,
size, and destination checks, and reports duplicate sources plus portable
case-insensitive destination collisions before any transaction exists. Every
destination remains editable and changing one invalidates the preview. Only a
fully clear, freshly reviewed batch can create the existing immutable journal
plan; execution still requires its exact transaction ID and revalidates every
path before copying.

Online enrichment is opt-in per library. Open Library publication suggestions
are available from Metadata review and disclose the exact title and optional
author before a request is made. Results always retain provider provenance and
must be selected and saved by the user; lookup never uploads a media file or
silently rewrites catalogue metadata. Requests are bounded, cached, and limited
to one concurrent Open Library call. Settings also provide write-only Hardcover
and TMDB credential fields, with provider choices restricted to compatible
library kinds. Each credential has an explicit connection test that can verify
either the saved server-side token or a newly pasted value without saving it,
reports actionable rejected/rate-limited/unavailable states, and lists the
shelves currently using that provider. Live Settings application preserves the
process-owned authenticated browser session while atomically replacing the
validated application handler; changing configured credentials still creates a
fresh security boundary. Provider tests use the shared response decoder, so an
expired login is distinguished from rejected credentials, rate limits, and
provider outages. TMDB-enabled video shelves can explicitly search for up to five
movie or television matches and review their year, synopsis, and provider source;
the bearer token never leaves the server. Movie shelves can choose title, year,
synopsis, and poster independently; those choices persist as reversible
Mochil catalogue overrides while same-origin, size-bounded artwork caching
keeps browsers away from provider hosts. Television and anime shelves apply the
same review model once per inferred show: the saved display title and poster do
not alter the original show key used for season grouping or filesystem lookup.
Enriched movies and shows expose their TMDB provenance and an inline, confirmed
removal action that restores scanner-inferred metadata without a browser popup.
Hardcover-enabled publication shelves can search its GraphQL catalogue from
Metadata review. The bounded two-step lookup preserves Hardcover result order,
adds title, authors, year, synopsis, cover reference, and source provenance, and
keeps its bearer token server-side. Title and author corrections still pass
through the existing explicit review decision before they become catalogue
overrides. A suggested cover can be reviewed, previewed through the same-origin
bounded artwork cache, and saved as a reversible catalogue override without
changing the source publication. Removing it restores the embedded cover.
Open Library suggestions now expose their returned cover through that same
reviewed, reversible workflow. Browsers load it only from Mochil' private
same-origin cache; the server accepts only the exact numeric medium-cover URL
shape on `covers.openlibrary.org`, rejects alternate queries, sizes, ports,
lookalike hosts, and cross-provider submissions, and leaves the publication
archive untouched.

## 2. Media foundation improvements

- Introduce shared favorites and history concepts only when multiple media
  modules prove the same abstraction is useful.

Bounded queues, keyed deduplication, cancellation, health reporting, and
bounded terminal history are already implemented. Running jobs can publish
validated counters and short public-safe status text through their worker
context; podcast downloads report throttled byte progress, and the bounded
terminal state retains their final reported value. The shared queue has an
optional versioned atomic state store. Podcast downloads use it so completed
diagnostics survive restarts and interrupted downloads return as failed work
with their last byte count, ready for the existing explicit retry control.
Malformed and oversized state is refused rather than partially trusted. Library entry resolves only
the selected shelf after presenting the shelf chooser. Its counts come from
constant-time store summaries rather than copying or enumerating every media
catalogue, and item or artwork APIs remain lazy until a shelf is selected.

## 3. Video and television

- Extend the implemented bounded trick-play previews if richer timeline
  navigation is needed.
- Add optional, reviewable metadata-provider artwork.

Direct play, HTTP ranges, movie and television grouping, probing, posters,
resume state, language preferences, optional HLS, bounded trick-play previews,
and the initial Jellyfin adapter are already implemented. HLS work is queued
through the shared bounded job pool and exposed in health/job diagnostics.

Local video sidecars now resolve with explicit catalogue scopes: show cards use
show-folder artwork, while episode cards prefer their season folder or common
`season01-poster` files at the show root. Every lookup remains size-bounded and
symlink-confined to the library, and generated frame posters remain the fallback.
Distant seeks serialize encoder replacement and wait for the canceled FFmpeg
process to exit before reusing its segment directory. Cache eviction protects
the stream being generated plus recently served sessions, preventing active
playback from losing its next segment while still reclaiming older entries.
Video shelves can now use Seafile through a narrow read-only list-and-range
capability. Remote scans are cancellable, extension-allowlisted, cache-backed,
and retain the last published snapshot on provider failure. Direct-play MP4,
M4V, MOV, WebM, and Ogg requests forward exactly one validated byte range and
stream a size-checked response without buffering the movie. Remote MKV/HLS,
subtitle extraction, preview frames, and generated posters remain unavailable
because they would require materializing large source files; the settings UI
states that boundary explicitly.

## 4. Podcasts

- Add richer per-episode download diagnostics only if aggregate retry proves
  insufficient.

Subscriptions, public-directory search, direct RSS entry, conditional refresh,
stable episode identity, streaming, resume and played state, downloads,
retention, and last-known-good behavior are already implemented. Directory
results expose normalized feed URLs and can be subscribed to without copying a
raw URL. Subscription management now shows each feed's last check
or refresh failure and exposes an explicit local retry state while confirming
that saved episodes remain available. Failed automatic downloads are counted
without exposing raw worker errors and can be re-admitted together through the
existing bounded queue. Each feed can independently auto-download a bounded
number of recent episodes and, when explicitly enabled, remove local downloads
after an episode has remained played for a selected number of days. Playback
state remains account-owned; the downloader receives only a cleanup decision,
preserving the boundary needed for future multi-user retention rules.
All server-side podcast requests use public-address-only dialing and revalidate
redirects, preventing feeds and their artwork or enclosures from reaching
loopback, private, link-local, or infrastructure metadata targets.

## 5. Audiobooks

Direct playback, downloads, chapters, artwork, resume, Continue Listening,
speed controls, natural series ordering, and the sleep timer are already
implemented. The scanner recognizes MP3, MP4/M4A/M4B, AAC, Ogg/OGA, Opus,
FLAC, WAV/WAVE, and audio-only WebM/WEBA. Embedded `author`, `albumartist`,
`grouping`, `series-part`, `reader`, `read_by`, `narrated_by`, and `performer`
aliases are normalized alongside the primary artist, album-artist, series, and
narrator tags. Fractional series positions remain numeric for correct novella
ordering, and chapter extraction remains bounded.
Audio and video progress records carry monotonic revisions. Browser players
serialize their own saves and use conditional updates. Routine passive
conflicts are reconciled quietly: equivalent checkpoints acknowledge the
authoritative revision, stale players yield until they catch up, finished media
cannot be reopened by background writes, and temporary reconciliation failures
remain retryable. Only a deliberate seek may claim an override. Android audio
and video apply the same policy, while durable audio seek ownership expires
after ten minutes so an old outbox entry cannot become a permanent rewind.
Focused browser, Go-store, and both Android-flavor unit suites pass across this
contract.
When direct decoding fails, local audiobooks and music now fall back to a
seekable AAC/fMP4 compatibility stream. The shared segmented engine uses a
bounded worker queue and disk cache, serializes distant-seek restarts, retains
the already-validated source descriptor, and exposes queue state through health
diagnostics. Compatible files remain direct-played, and downloads always retain
their original format.

## 6. Music

Tag and artwork probing, deterministic browsing, direct playback and
downloads, account queues, favorites, playlists, and bounded ReplayGain
normalization already have an implemented foundation. Track and album gain are
supported with peak-based clipping prevention and an R128 fallback. Music
queues use a dual-deck gapless transport that preloads at most one next track,
rejects tracks above 128 MiB from eager preload, and falls back to the ordinary
transition when the next deck is not ready.
Missing loudness metadata can be analyzed explicitly per shelf. One bounded
worker measures EBU R128 integrated loudness and true peak, preserves embedded
ReplayGain tags, revision-checks every result, and commits each completed batch
through one atomic catalogue-cache update. Source media is never modified, and
progress remains visible through shared job diagnostics.
The web Metadata workspace now selects music shelves without reusing the
publication repair table. Its bounded read-only quality report counts unknown
artists, exact duplicate track identities, and conflicting album tags within a
folder. It uses attached-cover facts already present in the probe cache and
checks a folder sidecar at most once only when an album has no known embedded
artwork; the inspection never starts a full-library media decode. Quality
grouping follows physical album folders and folds numbered CD/Disc subfolders
into their parent, preventing inconsistent tags from splitting the totals or
unrelated folders from producing cross-folder duplicate findings.
Confirmed publication repairs now return an authoritative file-impact receipt
instead of relying on browser inference. It distinguishes verified filesystem
EPUB updates, Calibre database records, and reversible catalogue overrides;
states that publication archives and `ComicInfo.xml` were not rewritten where
applicable; and reports whether the committed result was successfully observed
by the follow-up catalogue refresh. The real Calibre endpoint now supplies the
same typed counts used by the receipt, closing a prior server/test contract gap.
The web music shelf now opens new browsers on a bounded Discover destination.
Recently played, recently added, favorites, albums, and artists use short
independently loaded sections; Recently played has an account-scoped
twenty-track destination rather than an unbounded history request.
The persistent music queue now has a dedicated browse/edit view with play,
move, remove, and clear controls. Browser writes are serialized and coalesced
to the newest state, while queue/favorite/playlist stores commit memory only
after atomic persistence succeeds. Concurrent additions cannot lose entries;
startup decoding is size-bounded and rejects malformed saved references.
The native library now uses a compact Albums/Artists/Tracks menu. Album detail
uses one artwork hero and numbered track rows with artist and duration, avoiding
repeated artwork decoding for every track. The remaining staged first-class
music work is tracked in `MUSIC_EXPERIENCE_ROADMAP.md` rather than growing this
cross-product checklist into an implementation spec.
A bounded Discover view now combines recently played and recently added tracks,
favorites, albums, artists, and offline music without loading a whole shelf.
Recently played uses the account progress store, stays scoped to the selected
music shelf, and has a separate destination capped at twenty tracks. Track menus can alter
both the live and persistent queue through Play next and Add to queue.
They can also append a track atomically to an existing or newly named playlist.
Album and audiobook-series detail can download every missing track sequentially
with visible progress and without repeatedly rebuilding the offline index.
Playlists now appear directly in the native music browser with playback,
rename, ordered move/remove editing, and confirmed deletion. Whole albums and
series can be inserted next or appended to the synchronized and live queues in
one bounded operation.

## 7. Documents and wiki

Current file-backed vault parity is tracked in
[`WIKI_PARITY_AUDIT_2026-09-08.md`](WIKI_PARITY_AUDIT_2026-09-08.md).
Do not treat the legacy `/api/wiki` managed-store capabilities below as
implemented features of `/api/vaults` or the native Android vault client.
In particular, file-backed revision history/restore needs server support first;
it is not merely a missing Android screen. No legacy user data should be deleted
as part of removing old navigation or replacing the workspace.

- Design wiki collaboration and sharing with authorization, conflict handling,
  audit events, and retention rules.

The first native Android vault surface is implemented: collapsible folder/page
navigation, responsive phone and wide-screen layouts, native Markdown reading
with copyable fenced code, page/folder creation, and revision-conflict-safe page
editing. Its transport, state, hierarchy parser, Markdown parser, and Compose UI
are separate small modules. Native links/backlinks, rename/move, recoverable
trash, encrypted draft recovery, bounded tables and headings, and a bounded
encrypted read-only cache for up to 128 recent or explicitly mirrored Markdown
pages in each of the four most recently saved vaults are implemented.
The editor produces Obsidian-compatible headings, emphasis, strikethrough,
inline and fenced code, bullet/numbered/task lists, quotes, and links through a
compact horizontally scrollable strip; selection transforms are independently
tested and do not require a richer proprietary document format.
Vault caches use opaque, independent storage identities so visiting another
vault neither displaces nor internally mixes the first vault's offline pages;
the former single-vault cache migrates safely on first use.
The existing page menu can save vault pages offline with bounded four-request
concurrency and visible progress; it reports partial capacity honestly instead
of calling a truncated cache complete. Referenced images now mirror into bounded
no-backup app-private storage, retain valid copies through partial refreshes,
invalidate on revision changes, and report incomplete coverage. Revision
history/restore and non-image attachment mirroring remain outstanding. Local
and remote vault scanners now explicitly include PNG, JPEG, WebP, GIF, and AVIF
assets in the private vault tree so this rendering and offline pipeline works
with real Obsidian folders. Ordinary Documents shelves retain their narrower
format allowlist and do not become image browsers.

The read-only document scanner, bounded extraction and search, and safe reader
are implemented. Separately, the legacy managed wiki store implements its own
editor APIs, links, backlinks, history, restore, trash, and export. Its historical
notes/wiki workspace and reader exports must not be confused with the newer
file-backed Obsidian vault. Legacy managed pages carry a
portable nested folder path, appear in a compact folder tree, and retain their
folder through history, restore, persistence, and JSON export. Empty folders are
durable, and desktop pointer users can drag pages into folders, onto another
page as a subpage, or back to the root; the editable folder path remains the
keyboard and touch fallback. The web workspace
uses a dense edge-to-edge layout suitable for longer notes. Markdown, plain text, logs, CSV/TSV, JSON/JSONL,
YAML, TOML, and TeX are read and searched as bounded UTF-8 text; structured
formats render literally rather than being misinterpreted as Markdown. DOCX
and ODT files are identified explicitly
and their main document text is extracted through bounded ZIP/XML readers for
safe reading and content search. RTF control words, escaped characters, and
Unicode are decoded by a bounded parser that omits metadata, image, object, and
unknown optional destinations; malformed documents never reach the browser. The
wiki also supports permanent deletion after a seven-day trash hold, guarded by
the current revision and exact typed confirmation; the page and all revision
history are removed in one durable store update. The
Markdown reader supports headings, wrapped paragraphs, ordered and unordered
lists, blockquotes, fenced code, thematic breaks, and allowlisted links while
escaping raw HTML and refusing active URL schemes.
Document shelves can also use Seafile through a narrow list-and-read adapter.
Remote scans accept only the document format allowlist, reuse the last valid
cache on provider failure, and disable publication metadata hydration. Reads
use short-lived Seafile download URLs, a two-minute deadline, and a 128 MiB
spool limit before existing extraction, search, reader, or download paths see
the file. The adapter exposes no upload, rename, replace, or delete operation.

The web EPUB reader persists an explicit one-page or two-page paginated spread
choice. One page is the default; continuous scrolling remains independent.
The main web Settings page now offers browser-local library visibility switches
using the same compact model as Android. Hidden shelves are removed from the
shelf selector and Home continuation rails without changing or deleting server
configuration, and every shelf remains available in Settings for restoration.
The web Home heading is compact and media-neutral so Continue reading,
listening, and watching remain the primary hierarchy.
Settings categories remain sticky below the masthead while scrolling, and the
browser-visibility and server-library areas are collapsed accordions. Continue
Listening restores show or album artwork on its compact Home cards.

## 8. Integrated AI

This phase remains unimplemented and must stay optional:

- Configure providers and models explicitly.
- Require clear privacy and media-content sharing controls.
- Bound quotas, requests, stored data, and background work.
- Record provenance for generated metadata and require review before applying
  it.
- Store memories and annotations per account.
- Provide export and deletion controls.
- Keep indexing, browsing, reading, and playback fully functional when AI is
  disabled or unavailable.

## 9. Scale and multi-user readiness

- Separate authentication identities and administrator permissions.
- Isolate reading, playback, cache visibility, and settings by user and role.
- Expand large-catalogue benchmarks, malformed-input tests, and race coverage.
- Recheck that entering Library remains cheap as more media families and
  shelves are enabled.

The account boundary and migration order are audited in
`MULTI_USER_DECISION.md`. The owner-approved policy has no public registration,
promotes the current credential to administrator, and keeps global controls
administrator-only. The first migration slice is implemented: a bounded,
versioned account store hashes passwords, preserves the legacy administrator's
account key, assigns immutable IDs to new accounts, rejects malformed state,
and atomically supports create, disable, and password reset. Browser sessions
and Basic/OPDS authentication now attach that stable identity to each request;
personal reading, audio, video, collection, wiki, annotation, and external-title
handlers consume the request account instead of the configured global username.
Existing sessions are revalidated against the account store and stop working
when their identity is disabled. The environment credential remains an explicit
recovery path during migration. Administrator route gates and the account UI are
active. KoSync usernames and persisted random Jellyfin device tokens resolve the
same immutable account IDs, including disabled-account and cross-account progress
tests. Administrators can now download a bounded, password-free export of each
account's reading history, annotations, audio/video progress, queue, favorites,
playlists, collections, wiki, and external-library state. Permanent deletion is
disable-first, exact-confirmed, purges all personal stores and Android/Jellyfin/
KOReader credentials, and can be retried safely after an interrupted durable
write. The maintained `MULTI_USER_ISOLATION.md` gate maps browser, Basic/OPDS,
Android, KoSync, Jellyfin, migration, role authorization, and every account-keyed
store to cross-account and race-tested evidence.

Document browsing now uses snapshot-versioned opaque keyset cursors backed by
store-owned title and recent indexes. Ordinary pages no longer copy and sort
the complete document catalogue, stale cursors fail explicitly, and the prior
offset parameter remains available for compatible clients and search results.
Audio browsing now issues bounded opaque cursors tied to the exact filter and
sort plus the audio-catalogue, podcast, and account-playback snapshots. A
changed snapshot expires the cursor instead of silently skipping or repeating
tracks; offset requests remain compatible. Cursor codec cost is approximately
2.4 microseconds and 848 bytes per encode/decode round trip on the deployment
host.
Publication and series browsing now use the same snapshot discipline. Opaque
cursors bind the selected libraries, filters, search, and exact catalogue
generation; cross-view cursors fail validation and changed catalogues return an
explicit reload conflict. Existing offset clients remain compatible. Ordinary
single-shelf book pages use a stable ID-to-position index, while the cached
series index also owns representative-ID positions, so deep pages no longer
rescan or regroup the catalogue. Browser continuation prefers cursors and
automatically reloads after a generation conflict.
Persistent publication, audio, video, document, and document-text catalogue
loads now share a strict regular-file decoder. Startup reads stop at 128 MiB,
reject trailing JSON and malformed snapshots, and enforce one-million-entry
caps. The maintained 100,000-entry benchmark records decode time and memory so
future catalogue growth has a concrete regression signal.
External-library progress also has explicit malformed-record and simultaneous
writer coverage. Invalid chapter identities, locators, device IDs, fractions,
NaN, and infinity never enter state; two devices writing the same base revision
produce exactly one commit and one conflict whose response carries the
authoritative locator and revision.
Filesystem catalogue publication now advances its generation atomically when a
scan changes the published snapshot, allowing already-warm book and series
indexes to refresh immediately. No-op scans retain the generation and avoid a
needless 50,000-book rebuild. Dedicated benchmarks now cover full series-index
publication, filtered series regrouping, and publication-cache hits alongside
the existing warm and deep-page baselines.
Collection write requests are now capped at 16 KiB, while wiki writes retain
their bounded 2 MiB content allowance. Both use one shared transport decoder
that accepts exactly one JSON value; oversized bodies and trailing values fail
before feature validation or persistence. Collection and wiki handlers continue
to own their separate semantic rules. The same single-value invariant now
covers the existing bounded audio progress, queues, favorites and playlists;
video preferences and progress; podcast subscription and policy; reading and
KoSync state; metadata decisions; and Jellyfin session/progress writes without
changing their individual byte limits or validation policies.
Collections, wiki, reading state, audio/video progress, and podcast
subscriptions now also use the shared strict regular-file decoder at startup.
Their local limits range from 32 MiB for collections through 64 MiB for
progress state to 128 MiB for wiki and podcast history; oversized files,
non-regular files, malformed JSON, and trailing values are rejected before
state publication. Wiki startup additionally validates account, page, revision,
content, and bounded-history invariants so syntactically valid null or corrupt
records cannot reach normalization or request handling.
Library entry now has a maintained 100,000-book summary benchmark. Once the
derived index is warm, the count path takes about 59 nanoseconds and allocates
only its 64-byte result; its cost follows the number of configured shelves, not
the number of books. Audio, video, and document shelves retain their equivalent
constant-time count paths.

## Suggested next step

Seafile remains read-only across publication, document, and video shelves.
Android TV's compact playback dock now prioritizes the media itself: it shows
authenticated artwork, readable title/context and timeline, then only
play/pause, open Now Playing, and close. Previous/next remain in the full player,
reducing remote focus stops and preventing five transport buttons from crushing
the title. Home also stops consuming controller shoulder input when there is no
adjacent visible library. Both Android flavor unit/lint suites pass; connected
ten-foot confirmation remains part of the next batched APK because no ADB device
is attached to the development container.
Podcast storage no longer doubles as an RSS parser. Untrusted feed decoding,
description cleanup, stable episode identity, and date credibility now live in
`internal/podcast/rss.go`; subscription snapshots, refresh orchestration, and
atomic persistence remain in the 330-line store. Podcast and web suites pass,
and the module gate freezes the boundary.
Android mobile Home now keeps its continuation cards visually quiet at rest
while making swipe-to-dismiss legible: dragging a card reveals a restrained
`Remove from Home` surface, and long-press plus the accessibility custom action
remain available without permanent trailing icons. Direct and Store unit/lint
suites pass after the change; it remains batched for the next meaningful APK.
The current client-polish pass keeps compact controls visually quiet while
enforcing a 44 px browser touch floor and a 48 dp Android/TV interaction floor.
The web detector, rendered desktop/mobile/compact smoke suite, both Android
flavor unit and lint suites, module budgets, Go suite, and optimized APK
distribution gates pass after this adjustment. The direct-install updater is
now a direct-flavor implementation rather than shared code guarded by a
Boolean. Store builds compile an inert store-managed implementation and
store-safe resources, so debug and optimized store artifacts exclude APK
downloads, installer intents, installer MIME types, and unknown-app installation
guidance. The store-readiness gate enforces this source-set boundary and
inspects every supplied APK. It resolves the repository from its own location,
so packaging and CI checks behave identically from any working directory.
Podcast download recovery, server/local matching, retention, metered-network
policy, and automatic-download ownership are now isolated from Android podcast
screen state. Both distribution flavors pass unit tests and lint after the
extraction, and the architecture gate freezes the boundary.
The web media A–Z rail now derives its active letter from the newest visual row
that has crossed the sticky header. Uneven cover-card heights can no longer let
an older row steal the highlight while scrolling; focused mixed-height tests
and the complete 341-test frontend suite cover the policy.
Android TV library search now uses progressive disclosure: an existing query
keeps the compact Search control visibly active, opening it exposes the editor,
and submitting collapses the editor and restores remote focus to that control.
Returning to a searched shelf no longer spends 430 dp of header width on an
editor the viewer is not actively using. Direct and Store unit/lint suites pass;
ten-foot rendered confirmation remains part of the next connected-device pass.
Android music remote-catalogue transport is now separate from screen state.
Measured IO, parallel Discovery requests, group/track paging, artist lookup,
navigation, and favorite persistence live in `MusicCatalogueRepository`; the
view model keeps request-generation and presentation ownership. Both Android
flavor unit suites pass, and the architecture gate freezes that separation.
Web podcast and audiobook shelves now render missing artwork and drill-down
navigation with labelled SVG controls from the shared visual language. Their
new `audio-library.css` owner establishes a focused destination for subsequent
spoken-media extraction instead of growing the shared application stylesheet.
Web audio interaction routing is now separate from catalogue loading and
playback state. Queue mutations are captured exactly once before ordinary item
actions, while Discovery and playlist navigation retain their own delegated
path. Focused tests cover propagation and null-safe group selection; the main
coordinator is 2,425 bytes smaller.
Android PDF/CBZ synchronization no longer stores local revision state under an
ambiguous `library:item` key. Length-prefixed identities cannot collide when
either ID contains a colon, while first-open migration preserves existing
revision, exact-page, and offline-pending state. Both Android flavor unit and
lint suites pass after the migration.
The same collision-safe identity now covers exact local PDF/CBZ page and
finished state, plus Android audio favorites, queues, downloads, podcast
recovery, playback lookup, and UI keys. Legacy reader and download records are
migrated or normalized in place, so the hardening does not discard progress or
force media to be downloaded again. Both Android flavors pass unit tests and
lint, and module gates protect the shared identity boundary.
Video Media3 item and reconnect identities are collision-safe as well, keeping
retry budgets attached to the exact episode when IDs contain delimiters.
Publication, author, series, download, video, and TV continuation Compose keys
now use the same collision-safe library-scoped identity, preventing wrong-card
state or focus reuse in unusual but valid catalogues.
Android publication and collection network IO now lives in a focused
`PublicationCatalogueRepository`; `LibraryViewModel` remains the owner of
navigation, request generations, and screen state and has fallen to 420 lines.
Both Android flavor unit suites cover the refactored call paths, and the module
gate prevents transport from growing back into the coordinator.
Rendered web verification now proves the unified masthead and active library
controls remain anchored while catalogue content scrolls beneath them at
desktop, phone, and 320 px compact widths in both light and dark themes. This
protects the content-first single-shell behavior alongside the existing
overflow, touch-target, responsive-rail, and persistent-audio checks.
Settings categories now use the full workspace width independently of each
panel's readable measure. Wide metadata review no longer clips Accounts at the
old 760 px form boundary, while compact browsers retain a swipeable snapping
category rail. The authenticated render suite covers the deployed rule in both
themes and all maintained viewport widths.
Dashboard dismissal now follows progressive disclosure on fine-pointer web
clients: Continue Reading, Watching, and Listening cards reveal their remove
action on hover or keyboard focus, while touch devices retain the always-visible
44 px action. Reduced-motion users receive the state change without animation.
Large Android video catalogues no longer stop silently after the first 200
entries. Movie, show, season, and episode browsing now uses bounded 60-item
pages with total-count headers; mobile continues automatically near the end and
TV requests the next page through its virtualized grid. Show and episode cursors
remain independent, so a tablet two-pane view can extend either side without
resetting the other. The server only pages show responses when a client requests
it, preserving the unbounded response expected by older clients.
This Android paging slice ships in the batched 0.16.54 direct and Store builds
(version code 215). Both optimized APKs and the Store App Bundle complete R8,
resource shrinking, vital lint, flavor unit tests, and packaged store-readiness
inspection; the direct APK remains signed by the installation's persistent
self-hosted update key.
Web movie browsing now keeps only three 120-item pages mounted at once. Measured
top and bottom spacers preserve scroll position, automatic continuation loads
forward, and an independent previous-page sentinel restores recycled cards on
back-scroll. This bounds browser memory without adding visible pagination chrome
or applying destructive recycling to expanded television season trees.
Ordinary web document shelves now have the same bounded, bidirectional behavior
in both cover and list layouts. Versioned forward cursors recover automatically
after a concurrent scan, while back-scroll uses stable offsets to restore a
trimmed page. Obsidian-compatible vault trees remain complete and unwindowed so
nested folders, the active page, and editor state are never recycled.
The first-class web music surface now has a maintained rendered and code-level
quality audit in `WEB_MUSIC_AUDIT.md`. It scores 20/20 with no accessibility,
performance, responsive, theming, or implementation-integrity failure. Compact
destination rails retain a clear gutter before the fixed View action without
adding a second header row or shrinking touch targets. The authenticated render
suite verifies desktop, phone, and 320 px compact widths in dark and light
themes alongside frontend, Go, module-size, deployment-health, and asset-version
checks.
The web audio coordinator no longer manufactures filter, destination, sorting,
or chapter controls during startup. Those elements now live in the existing
audio library and player templates, preserving one accessible source of truth
for the UI while leaving the coordinator responsible only for state and
behavior. A module-boundary regression, all 360 frontend tests, the complete Go
suite, the module-size gate, and the Impeccable detector pass after the
extraction.
Its media-specific paging presentation is now pure and independently tested as
well: grouped music counts, bounded track-window offsets, finite destinations,
load-more wording, and podcast/audiobook Play All eligibility live in
`audio-render-policy.js` rather than a dense renderer expression. This keeps
future spoken-media or music refinements from silently changing another media
kind's controls.
Podcast and audiobook surface selection has also moved into their existing
family view owner. Overview, focused series/show, author, cover/list, filtered
feed, and empty-state choices no longer expand the shared coordinator. The
coordinator is now 218 lines / 31,834 bytes, with a tightened architecture
budget and explicit boundary checks rather than an arbitrary line-count claim.
Play Store removal verification now covers the artifact that is actually
uploaded, not only installable APKs. The store-readiness gate automatically
discovers optimized Store AABs and rejects retired extension identities,
personal deployment hostnames, and direct-installer behavior in both APK and
bundle payloads. The current direct APK, Store APK, and Store AAB all pass.
The independent scale-hardening pass now covers large-catalogue query,
publication, cache, decode, and Library-entry benchmarks; strict bounded state
loading and mutation decoding; malformed-state regressions; and the complete
race suite. The remaining slices require an owner choice: richer video preview
or podcast diagnostics based on observed need, a metadata artwork provider and
privacy boundary, or the account/role policy needed by wiki collaboration and
multi-user isolation. Integrated AI remains the largest untouched phase and
should stay last, after privacy, quota, provenance, and account-isolation
contracts are defined.
The native Android TV surface now has a maintained code-level quality audit in
`ANDROID_TV_AUDIT.md`. It scores 19/20: remote navigation, stable focus memory,
ten-foot layout, lazy rendering, paging, and modular ownership pass. The final
point is deliberately reserved for the next batched APK's physical 720p/1080p
overscan, directional-focus, focus-scale, and motion verification rather than
being inferred from source alone.
The connected Android handset shell now has a maintained navigation audit in
`ANDROID_MOBILE_NAVIGATION_AUDIT.md`. Its first P1 is remediated: Settings,
Saved, and Sources remember whether Home or Library opened them, and both
system Back and visible Back controls return to that primary origin instead of
always discarding the library context. Direct and Store unit/lint suites cover
the bounded history policy. Auxiliary roots also retain visible active context.
Compact windows use the lazy snapping destination rail; windows at 840 dp and
above use a virtualized, restrained Material permanent drawer and suppress the
duplicate in-screen rail. The source-level audit is now 19/20. Its last point is
reserved for the next batched APK's physical tablet/foldable verification in
light/dark themes, larger font scales, split screen, gestures, and system
insets.
The Wiki controller's first concealed monolith boundary has been extracted by
responsibility rather than line count. `wiki-markdown.js` now owns the bounded
Markdown dialect and escaping contract, with hostile HTML, Wiki-link attribute,
fenced-code, empty, and malformed-content tests. `wiki.js` retains navigation,
editing, history, trash, and drag/drop orchestration and is 984 bytes smaller;
370 frontend tests, the module-size gate, and the Impeccable detector pass.
Wiki page presentation is separated now as well. Active/deleted status,
retention availability, action markup, Markdown content, and backlinks live in
`wiki-page-view.js`; malformed timestamps no longer expose `Invalid Date`, and
backlink IDs cannot escape their HTML attribute. The coordinator is down to
10,439 bytes while retaining state and event ownership. All 374 frontend tests,
the module-size gate, and the Impeccable detector pass.
Wiki history rendering is now independently owned by `wiki-history-view.js` as
well. It handles safe revision labels, malformed timestamp fallbacks, escaped
diffs, unchanged revisions, and bounded oversized comparisons, while restore
network mutations remain in the coordinator. `wiki.js` is now 9,184 bytes and
all 378 frontend tests plus architecture and design gates pass.
Wiki editor markup and failure presentation are now independently owned by
`wiki-editor-view.js` and `wiki-errors.js`. Hostile persisted form values stay
escaped, and malformed save, move, purge, or folder-creation rejections now
produce recoverable messages instead of secondary JavaScript errors. The
remaining coordinator is a cohesive state-transition, HTTP-mutation, and event-
wiring owner, so the Wiki split stops here rather than chasing a smaller line
count. All 384 frontend tests and the architecture/design gates pass.
Video progress now shares audio's idempotent stale-clock contract. When two
clients report the same finished state within 250 ms, a stale revision is
acknowledged without replacing the newer device, timestamp, position, status,
or revision. Real rewinds, finished-state changes, and materially different
positions still return a conflict; deliberate seeks retain their explicit
override. This removes a needless GET/retry cycle and routine conflict surface
from web, Android mobile, and Android TV playback. Focused, full Go, vet, and
race tests pass.
Android 0.16.65 (version code 226) completed R8, resource shrinking, vital lint,
signing, and package-level Store-readiness inspection for its Direct APK, Store
APK, and Store App Bundle before being superseded by the connection hotfix. It added bounded
JSON transport coverage, richer native Wiki formatting, indexed Wiki image
resolution, and status-only Home dismissal that preserves newer remote playback
progress. Physical compact-phone, wide tablet or
foldable, and Android TV confirmation remains the next feedback gate.
Android 0.16.66 (version code 227) is the deployed connection hotfix. It moves
canonical redirect validation into one network boundary shared by Wiki and
audio catalogue reads, allowing bounded same-server API canonicalization while
refusing login pages, cross-host credential forwarding, HTTPS downgrades, and
unbounded chains. Audio mutations remain deliberately non-redirecting.

The next Android batch also isolates the bounded audio and video progress
revision caches by paired device credential. Disconnecting and pairing another
account on the same server can no longer reuse the previous account's cached
revision or passive-progress decision; rotated tokens for the same device keep
the useful cache. Direct and Store regression tests cover both boundaries.
The same account-boundary audit found and corrected a stronger issue in the
encrypted Saved-OPDS cache: device-local titles and the current account mirror
now have separate scopes, legacy rows migrate only to local storage, and pending
titles from an earlier pairing are never uploaded after an account change.
Authenticated standalone sources are isolated at the disk-cache boundary too.
OPDS artwork and WebDAV page, index, and stream-reference caches use random,
non-secret generations stored with encrypted connection settings. Editing an
address or credential invalidates old derivatives even if Android terminates
the app before background cleanup; renaming a source keeps its cache intact.
Native Wiki offline pages, mirrored images, and the one encrypted unsaved draft
are now paired-account scoped too. Older unowned app-private derivatives are
discarded on first vault access and rebuilt from the authenticated server;
canonical Obsidian files remain untouched.
Durable audio progress checkpoints now carry the same paired-account boundary.
Restarting playback cannot replay a previous account's pause or seek through a
new credential; legacy unowned outbox rows are discarded before synchronization.

The native Wiki state owner now delegates dependent server operations to a
tested repository. This keeps cancellation and current-page checks in the
ViewModel while centralizing tree/page sequencing and correctly distinguishes
“saved, revision reload failed” from “write failed.” It is an architecture and
stability refinement only; external Obsidian vault files remain canonical and
the existing Android Wiki UI is unchanged.

Android 0.16.68 was physically verified on a connected handset: the native
Wiki opens without the former 303 failure and the indexed audiobook shelf is
visible. Android 0.16.69 (version code 230) is now deployed with the subsequent
account-isolation, cache-generation, progress-outbox, and Wiki repository work.
Its Direct and Store APKs and Store App Bundle completed unit tests, lint, R8,
resource shrinking, signing, module-size enforcement, Store-readiness checks,
and full server tests before rollout. The running container is healthy and its
hosted APK is byte-identical to the verified Direct artifact. Android TV
overscan and remote traversal remain deferred until the TV device is available.

The web audio catalogue now delegates ordinary music-track and spoken-media
page requests to a focused transport owner. Cursor/offset selection, media-
specific page sizes, encoded search/filter values, abort propagation, and
untrusted pagination headers are independently tested, while the UI coordinator
keeps request generations, rendering, and its bounded virtual window.

Android mobile now scopes queue and playlist observation to the Library and
expanded-player surfaces that consume it. Editing a queue or refreshing
playlists no longer recomposes the connected application root, Home, Settings,
or the navigation shell; closing the expanded player also removes both
subscriptions entirely while the lightweight mini-player remains active.
Mobile and TV library-directory consumers now share distinct shell and
visibility projections as well. Catalogue paging and search state remain inside
Library, so they cannot invalidate Home, Settings, or the persistent navigation
surface merely because the selected shelf is loading more content.

Cross-device reading conflict suppression now uses semantic location evidence
in addition to renderer fractions. Matching CFIs, KOReader XPointers, and
contextual text anchors are idempotently acknowledged without moving either
reader or replacing the authoritative record, preventing screen-size reflow
from producing a routine “changed elsewhere” state at the same sentence. Fixed-
layout PDF/CBZ progress remains exact-page only, and genuinely divergent or
ambiguous locations still require review.

Saved OPDS publications now receive the same passive-reader protection. Their
existing bounded Readium locators are compared by resource, selected text, and
surrounding context, so a stale device at the same EPUB passage acknowledges
the authoritative revision even when reflow changed its percentage. Different
or context-free locators still conflict, and the authoritative record is never
replaced by an acknowledgement.

The responsive web shell now defers the complete Wiki/document controller
graph until a document shelf is actually selected while mounting its small DOM
contract up front. Settings and metadata administration now share the same
deferred route boundary; their surface is inert and reports a loading state
until both bindings and server data are ready, preventing fast clicks from
being lost during initialization. A repeatable authenticated browser benchmark
measured Home falling from 134 to 54 script requests and from 538,799 to 165,898
transferred script bytes (about 69% less JavaScript). Home retains lightweight,
bounded Continue listening and Continue watching controllers; complete music,
podcast, audiobook, video catalogue, metadata, HLS, and player graphs load on
their first relevant action. Cold Music initialization remained below 600 ms
in the measured runs. Route-owned styles now load through the same boundary and
resolve before their controller renders: initial stylesheets fell from 21 to 9
requests and from 181,729 to 115,452 transferred bytes (about 36% less CSS).
Measured initial CLS remained 0 and stayed 0.0437 after opening Music, below the
0.1 Core Web Vitals target. Generation guards prevent a deferred shelf from committing
after the user navigates away. The root coordinator fell back to 377 lines
after focused route owners were extracted. The full desktop/mobile/compact
dark/light browser matrix also verifies lazy Wiki, Settings, audio and video
loading, playback navigation, and persistent audio.

The current Android TV source audit confirms that catalogue grids remain lazy,
item identities are stable, scroll/focus history is bounded, Home observes only
the library-shell projection, and the playback clock recomposes below the dock
rather than invalidating the application root. One hardware-correlated layout
defect was corrected: the Library back/browse rail now keeps its remote targets
24 dp away from the physical left edge instead of placing them 2 dp from it.
The TV regression suite and Direct lint pass. Physical overscan and traversal
verification remains the next TV feedback gate; this source-only fix will ship
with the next meaningful Android batch rather than forcing a one-line APK.

Native Wiki vault browsing now builds one immutable, naturally ordered tree
index per server entry revision on a background dispatcher. Folder expansion,
collapse, and local filtering reuse that index instead of regrouping and
sorting the entire Obsidian vault on every interaction. The index remains a
small dedicated owner outside the screen and ViewModel, attachments stay
hidden from the page tree, collapsed pages remain searchable, and focused plus
complete Android tests, lint, and module-boundary enforcement pass.

Store-readiness was re-audited against the active Go, browser, Direct Android,
and Store Android source roots plus the optimized Direct APK, Store APK, and
Store App Bundle. The executable extension/Suwayomi/Tachidesk family remains
absent from runtime code, dependencies, manifests, and packaged strings; Store
packaging also remains free of the direct self-installer, broad storage/package
permissions, and personal deployment hostnames.

The authenticated browser benchmark now enforces configurable regression
budgets instead of only printing measurements. Its default gates cover Home
JavaScript/CSS transfer, DOM size, Music cold-navigation time, post-Music
JavaScript/CSS and DOM size, and CLS for both surfaces. The current deployed
run passed at 165,898 bytes Home JavaScript, 115,452 bytes Home CSS, 617 nodes,
zero initial CLS, 268 ms cold Music navigation, 333,609 bytes post-Music
JavaScript, 157,684 bytes post-Music CSS, 1,126 nodes, and 0.0437 CLS. All 425
frontend tests and module-boundary enforcement pass.

The web audio boundary now defers podcast subscription/download management
until a podcast shelf is actually opened. Cold Music no longer downloads or
initializes that unrelated controller, reducing its first-open graph from 102
to 101 scripts and from 333,609 to 324,406 transferred JavaScript bytes in the
deployed authenticated benchmark. Podcast selection was separately verified to
load the controller on demand, and the full six-viewport dark/light browser
matrix still passes with audio persisting across navigation. All 426 frontend
tests and module-boundary enforcement pass.

Podcast episode paging and audiobook hierarchy/paging now use stable lazy
catalogue facades as well. Music keeps the same interaction bindings but does
not download either spoken-media state owner; generation checks discard a
catalogue module that finishes loading after its route was reset or abandoned.
The deployed cold-Music graph is now 100 scripts and 319,481 transferred
JavaScript bytes, down from 102 scripts and 333,609 bytes before the spoken
split, while navigation measured 261 ms. The rendered browser suite opens an
audiobook series and a podcast show through the deferred owners, verifies their
items, and returns to each root. All 428 frontend tests and the updated
module-boundary gate pass.

Audiobook chapter parsing and chapter-panel DOM bindings now sit behind their
own stable lazy facade too. Ordinary songs pay only for the small facade;
opening the first chaptered audiobook loads and renders the full controller,
while closing or replacing the item invalidates a pending load. The deployed
cold-Music graph remains 100 scripts and fell to 318,366 transferred JavaScript
bytes with a 267 ms measured open. An isolated audio/DOM contract verifies the
first deferred chapter list without mutating personal listening progress. All
430 frontend tests, the full browser matrix, and module boundaries pass.

Android 0.16.70 (version code 231) is now deployed as the next batched native
release. Its changelog contains only this release: safer Android TV library
edge spacing, one background-built Wiki vault index per entry update, narrower
queue/playlist recomposition, and compact Home library projections. Direct and
Store APKs plus the Store App Bundle completed the full unit suite, lint, R8,
resource optimization, signing, Store-readiness inspection, and architecture
gates. Both APKs use the preserved certificate fingerprint; the Store package
does not request package installation. The authenticated web download reports
0.16.70 and is byte-identical to the verified Direct artifact (SHA-256
`88b7f3dcdc009fe652f20b3353a7d0650055057c1f40d608555b26e153851615`).

Web loudness analysis is now progressively disclosed behind a dedicated lazy
facade. Opening Music or Audiobooks exposes a concise `Check loudness` action
without downloading the analysis controller, polling, or making an analysis
API request. The first explicit check loads the tool and then presents a clear
`Analyze missing`, `Analyzing…`, or `Analysis complete` state. On the deployed
instance this reduced cold Music from 141 to 140 resources and from 318,366 to
316,858 transferred JavaScript bytes while retaining a 267 ms navigation and
0.0437 CLS. All 432 frontend tests, Go web tests, module-boundary checks, and
the desktop/mobile/compact dark/light browser matrix pass; playback still
persists across navigation.

The optional web Track information surface now follows the same progressive
disclosure rule. Cold Music retains only a small pure formatter for the format,
bit depth, and sample-rate facts shown by the player; the dialog controller and
its dedicated stylesheet load on the first Track information action. The live
cold-Music graph fell again to 314,812 transferred JavaScript bytes and 155,373
CSS bytes (from 316,858 and 157,684 immediately before this split), with 288 ms
navigation and unchanged 0.0437 CLS. The browser matrix explicitly opens and
validates the deferred dialog in every supported viewport/theme profile. The
Impeccable-triggered ownership audit confirmed the route bootstrap remains a
small loader rather than absorbing dialog state. All 433 frontend tests, Go
web tests, and module boundaries pass.

The web audio coordinator no longer owns the long list of route-specific
control mutations. A focused surface controller now atomically exposes only
the controls relevant to Music, Podcasts, or Audiobooks and dismisses transient
playlist creation on every route change. Music initialization was expanded
from a compressed multi-branch statement into an auditable flow with the same
generation guards, and opening the player computes source facts once. The
deployed desktop/mobile/compact dark/light matrix passes across audio routes
with persistent playback; the performance budget remains healthy at 295 ms
cold Music navigation, 315,958 transferred JavaScript bytes, 155,373 CSS bytes,
and 0.0437 CLS. All 435 frontend tests, Go web tests, and architecture gates
pass.

Cross-device media progress parity was re-audited across the server, browser,
and native clients. The existing passive policy remains intentionally
monotonic: stale players cannot rewind or unfinish newer progress, equivalent
clock samples acknowledge the authoritative revision without writing, and only
a one-shot explicit seek may move backwards. One native inconsistency was
fixed: Android video checkpoints now include the bounded paired-device ID just
like Android audio and both web players. Server coverage proves that attribution
survives persistence and is returned with the authoritative video position.
The complete audio, video, reading, and web server suites pass, as do the full
Direct/Store Android unit and lint suites. This native-only fix will remain in
the next meaningful Android batch rather than forcing another immediate APK.

Native Wiki links now support the creation workflow expected from an
Obsidian-compatible vault. An existing `[[link]]` still opens immediately; a
missing link now offers a contextual confirmation instead of becoming a dead
error. Simple names are created beside the current page, explicit folder paths
remain vault-relative, and empty or traversal paths are rejected. Read-only
and offline copies never offer a write action. Resolution policy, presentation,
and mutation remain in separate focused modules rather than enlarging the Wiki
screen. Focused Wiki tests plus the complete Direct/Store Android unit and lint
suites pass. This change is queued for the next batched Android release.

Android offline-audio state now comes from one validated storage snapshot per
refresh instead of independently walking the same local and document-provider
files for IDs and track rows. Snapshot validation runs on the IO dispatcher,
so opening Music no longer performs that external-storage work on the UI
thread. Download completion is also library-scoped: a late result cannot write
the previous shelf's offline rows into a shelf opened meanwhile. The complete
Direct/Store unit and lint suites pass. This performance/stability improvement
is queued with the next native release rather than producing a premature APK.

The web Music cold path no longer downloads podcast and audiobook rendering
templates. Music favorites retain a small family-owned filter/sort module,
while the 6 KB spoken presentation module loads only when Podcasts or
Audiobooks is opened. Compared with the former shared 6,884-byte module, the
cold graph now carries 1,056 bytes of Music policy and lazy-facade code, a
5,828-byte raw JavaScript reduction. A route-generation guard prevents a slow
deferred import from reclaiming the screen after the user switches libraries.
All 437 frontend tests, web server tests, Store-readiness checks, and module
boundaries pass. Cache generations were advanced and the healthy local
container now serves the updated controller (`v8`) and audio library (`v125`).

The September 7 restart incident was traced to a private config.json owned by
UID 1000 while the application runs as UID 65532. Startup now repairs ownership
of this regular config file without changing its permissions or traversing
media. The isolated Compose install test reproduces this mismatch and verifies
recovery with mode 0600. The fix is deployed and the actual container is healthy.

Offline-audio refresh ordering now has a dedicated owner. A startup-wide scan
cannot replace a subsequently opened shelf, and older refreshes of the same
shelf cannot overwrite newer download results. Download completion uses this
same refresh boundary. Tests deliberately complete requests in reverse order,
including a storage operation that ignores cancellation; both Direct and Store
audio test suites and the architecture gate pass. Queued for the next APK.

Offline audio batch progress now updates state on the main dispatcher and is
scoped to the library visit, preventing late background updates from overwriting
navigation (including leaving and reopening the same shelf). Cancellation is
checked between files for downloads and removals. Offline snapshot failures
preserve existing rows and report a retryable storage-access message instead
of escaping the refresh coroutine; cancellation is not treated as an error.
Direct and Store audio unit tests cover batch cancellation, refresh ordering,
storage failure and retry, and refresh cancellation. Both suites and the
module-size gate pass. These changes remain queued for the batched APK release;
no mobile layout changes or new APK were made.

Batch audio download/removal planning now validates one storage snapshot on
the IO dispatcher instead of calling contains (and revalidating the complete
store) for each track on the UI thread. Selection uses indexed identities,
including stable podcast feed/episode aliases across recreated libraries.
Planning rejects obsolete library visits, handles storage failures, and guards
repeated batch taps while selection is pending. A 1,000-track regression test
asserts one snapshot, zero per-track contains calls, and a non-caller storage
thread. Direct/Store audio suites and the module-size gate pass. No wall-clock
speedup is claimed; APK remains queued.

Offline audio transfers no longer hold the index monitor for the duration of
the HTTP download. A dedicated mutation gate serializes downloads/removals,
then locks the index for publication, allowing reads of existing downloads
during network transfer. Concurrency tests verify readable old state while a
transfer is held, publication before a queued removal, and lock release without
publication after transfer failure. Direct/Store audio tests and the module-size
gate pass; queued for the next APK. Remaining work: external-folder mirroring
still holds the index monitor during publication, and playback lookup still
validates storage synchronously. Do not call the entire offline path non-blocking
until these remaining access paths have been addressed and device-tested.

Playback and single-track membership now match cached metadata before validating
storage, rather than validating every offline file first. The returned local
path itself must pass directory containment/readability checks; a valid external
URI no longer permits an unrelated invalid local path to be selected. Tests
cover 1,001 entries with only one storage resolver call, absent identities with
zero calls, and unreadable podcast copies followed by a readable recovered
alias. Both Android audio suites and the module-size gate pass. Queued for the
batched release. Matching-file validation remains synchronous, so this reduces
unnecessary IO but does not establish a fully asynchronous playback path.

Offline audio removal now independently validates the local path before deleting
anything; a valid external URI cannot authorize deletion of an invalid local
path in the same index row. Shared local-file policy rejects directories,
outside paths, and symlinks resolving outside the private audio folder while
allowing missing in-folder files during cleanup. Filesystem tests exercise
these cases, including actual temporary symlinks. Both Android audio suites
and the module-size gate pass. This is queued for the next batched APK; no
user media was deleted during this work.

New private audio download filenames now use SHA-256 of the complete track
identity. The former punctuation substitution and 180-character truncation
could map distinct tracks to the same target/partial file; Unicode names could
also exceed byte-based filesystem limits. Tests cover punctuation collisions,
long shared prefixes, and bounded deterministic ASCII output. Existing completed
downloads still use their persisted paths; legacy partial files are not migrated
because their old filename cannot reliably prove track identity. Both Android
audio suites and the module-size gate pass. Queued for the batched APK.

Failed external-folder copies now attempt to discard only their newly created
document, avoiding abandoned partial media being mistaken for completed files
during later recovery. Existing documents and the private offline copy are not
removed. Cleanup failure preserves the original copy error. Tests cover success,
creation failure, interrupted writes, and cleanup failure; these tests plus the
audio suites pass for Direct/Store, as does the module-size gate. Providers can
still refuse cleanup, so this is best-effort recovery, not a transactional SAF
guarantee. Queued for the next APK; no user documents were deleted by this audit.

Broader native verification completed: 548 unit tests per Direct/Store variant
passed after the offline-storage work, and both lint tasks completed successfully.
The Store-readiness script also passed. Subsequent Wiki resolver fixes now give
explicit vault paths precedence over duplicate labels, avoid resolving missing
folder-qualified links to another folder, exclude attachments from page lookup,
and handle uppercase .MD suffixes. Both Wiki test suites and the module-size
gate pass after those edits. Lint was started before the Wiki edits, so it is
not claimed as a fresh lint verification of them. No APK or UI layout change.

Wiki save acknowledgements now preserve text entered after submission, update
the saved baseline/revision, keep editing active when newer text exists, and
reschedule encrypted draft persistence against that revision. Repeated Save
calls are rejected while saving. Page navigation resets its own saving state;
a previous page's save response/failure no longer clears another page's state.
Reducer tests cover newer text, unchanged submissions, and different page/path/
vault results. Both Direct/Store Wiki suites pass. Same-page leave-and-return
request ownership and other mutation paths still need a broader lifecycle audit.
Queued for the next APK without layout changes.

Wiki mutation callbacks now capture a page-session token invalidated on vault
open, page selection, and page close. Save/task/create/move/trash success and
failure responses from earlier visits are ignored even after reopening the
same page ID. Create/move/trash also reject repeated mutations while saving;
closing a page clears its saving state. Server mutations are not rolled back
by navigation. Session invalidation tests and both Wiki suites pass, along with
the module-size gate. Page/tree read callbacks and refresh/edit races remain
separate follow-up work. Queued for the next APK.

Wiki refresh completion now rechecks live editing/saving state and changes to
the page baseline/draft, not just whether the draft was dirty when refresh
started. New edits and newer save revisions survive delayed refresh responses.
Remote deletion retains the active draft/selection with an explanation. Refresh
callbacks also respect the page-session token. Five reducer tests cover these
races and unchanged-page refresh; Direct/Store Wiki suites and the module-size
gate pass. Queued without layout changes. Initial vault/page load ownership
remains to be audited separately.

Initial vault and individual page loads now capture the page-session token too.
They recheck after cached-storage reads and remote responses, including the
additional draft/asset awaits during vault opening. Obsolete loads cannot
publish a page, clear another selection's draft, or start stale backlinks after
navigation. Failure callbacks use the same ownership check. Both Android Wiki
suites and the module-size gate pass. These are source/unit-verified lifecycle
guards; real-device rapid-navigation testing is still needed with the next APK.

Compound Wiki requests now check cancellation between dependent blocking client
calls. Save reload cancellation is propagated rather than wrapped as a reload
failure; real reload errors still distinguish a successful write from an
unsaved failure. Tests exercise cancellation after a completed write (no second
request) and cancellation during reload (original cancellation retained).
Direct/Store Wiki suites and the module-size gate pass. In-flight blocking HTTP
calls are not forcibly interrupted by this change; it prevents subsequent work.

Server audio/video status patches now return existing progress unchanged when
the requested status/duration/position already matches. Repeated actions no
longer advance revisions/timestamps, invalidate audio snapshots, or rewrite
the progress JSON. Regression tests verify that a subsequent active-player
checkpoint still accepts its revision and that no-op status retries succeed
without a writable progress-file destination. Audio/video tests pass with the
race detector; web HTTP tests and the module-size gate pass. Source change is
not yet deployed to the running server.

Deployment follow-up (September 7, 14:02 UTC): full `go test ./...` passed,
the local server image was rebuilt with single-CPU compilation and deployed
through Compose. The new container is healthy, `/healthz` returns `ok`, and all
10 configured libraries loaded from cache. The existing malformed Regressor
EPUB warning remains unrelated and its file was left unchanged. Idempotent
audio/video status patches are now live. Android changes remain queued.

Android update signing preflight: the persistent optimized keystore certificate
matches the existing direct optimized APK. Optimized packaging now fails if
that key is missing instead of silently substituting a machine-generated debug
identity. The positive verification task passes; an explicit nonexistent-key
test fails with recovery instructions. Dry-run task graphs confirm the guard
precedes Direct/Store optimized packaging. Store-readiness and module-size
checks pass. No new key or APK was generated in this preflight.

Android 0.16.71 / code 232 is staged in Gradle and the server distribution
metadata, with only the current batch's fixes/performance notes. The updated
distribution test verifies this release's content and rejects the previous
release's notes. Web tests pass; the full Android unit run reports 563 passing
tests per Direct/Store variant. Final lint is running before optimized
packaging. Do not deploy this new server metadata until the matching APK is
packaged and its signing certificate/version verified. The live server and
served APK remain on the previous Android release at this point.

Release-context audit additionally found that Git ignored the persistent
Android signing key but Docker's build context did not. `.dockerignore` now
excludes the signing directory, keystore properties, and .keystore/.jks files;
Store-readiness enforces those exclusions. The check and shell syntax validation
pass. Existing local Docker build cache was not deleted; older build-stage
cache may contain previously copied signing material. The final runtime image
copies only the server binary/data and entrypoint, not the source tree. Keep
local build caches private. Final Direct lint passed; Store lint remains running.

Release follow-up: both final lint tasks passed. Direct optimized Android
0.16.71 / code 232 completed, passed APK signature verification and packaged
Store-readiness checks, and matches the previous certificate (SHA-256
5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246).
The matching server image was deployed at 14:23 UTC; health is OK and all ten
libraries loaded. The mounted download is byte-identical to the verified APK:
SHA-256 7d0387eb3f8afa1179a3b386dd97bb117f760bd9c5bdc3620364e82c5dee6abf.
The Store optimized build remains running and is not yet verified. Device
installation, TV focus/TalkBack, and rapid Wiki navigation checks remain pending.

Store optimized packaging subsequently completed successfully (combined build
7m59s). Both APKs report 0.16.71 / 232 and verify against the persistent private
certificate. The packaged Store-readiness check passes for both artifacts;
the Store permission dump contains no package-install or broad-storage access.
Store APK SHA-256:
f0fcbb7cc8352fc4ace03ad70148345155d7bfb48ead9bb64a4bcb26b33b8bdd.
This is a privately signed optimized verification build, not a public Play
Store submission. The live deployment continues serving the Direct APK and
remains healthy with zero restarts. No build jobs remain from this release.

Post-release server audit: audio/video stores explicitly reject NaN progress
times/status durations, and audio normalizes NaN playback rates through the
existing normal-speed fallback. Range checks already reject infinities; tests
cover NaN and both infinities without altering stored progress/revisions.
This protects internal store callers as well as the existing JSON boundary.
Audio/video/web tests pass with the race detector; module-size gate passes.
Queued for a later server deployment; the published Android release is unchanged.

Checkpoint follow-up: identical normalized audio/video checkpoints now skip
persistence and preserve timestamps/revisions (and the audio cache version).
This applies to unconditional, revision-checked, and explicit-seek writes after
the existing conflict checks. Device, status, duration, speed and even 0.01-second
position changes still persist. Regression tests use an unwritable destination
directory to prove no-op checkpoints do not attempt a write; separate tests cover
each meaningful state change. Audio/video/web race tests and the module-size
gate pass. This is queued server work, not yet deployed; Android is unchanged.

Reading-state follow-up: repeating an already-applied Home dismissal now
preserves the reading revision and skips persistence/whole-store cloning.
Returned positions remain defensively copied. Put now builds its rollback
snapshot only after revision checks, avoiding full account/history copies for
stale conflicts and equivalent acknowledgements. Rejected initial writes no
longer leave empty accounts behind. Tests cover dismissal retry without a
writable state destination, response isolation, unchanged history, active
reader continuation and rejected initial writes. Reading/web race tests and
module-size checks pass. Queued for server deployment; no mobile UI changes.

Measured the avoided rollback-snapshot cost for stale reading acknowledgements
at 100 and 5,000 entries with five snapshots each. Reproducible benchmark and
scope/limitations are in SYNC_PERFORMANCE_2026-09-07.md. The full Go regression
suite passes with the queued server changes. No APK rebuild or deployment in
this verification pass.

Deployment follow-up (September 7, 14:42 UTC): queued audio/video validation
and checkpoint idempotency plus reading retry/snapshot optimizations are live.
Image a1b9cc0224910b228dc3c17ad3b70d89e1d5ed009f0a3837761f210538abe313
passed isolated Compose startup and restored-config/missing-library recovery.
Only the opdshelf service was recreated; live health passed, zero restarts,
all ten libraries loaded. Existing malformed EPUB warning remains isolated.
Temporary smoke-test resources were cleaned up; no user data removed. Android
remains 0.16.71 / 232 with no new APK or UI changes.

Android Wiki load-race follow-up: online open/select results now pass through
WikiLoadState, preserving edits, recovered drafts, in-flight saves and newer
saved pages that appeared while a cached page was loading. The original
revision remains the save-conflict baseline rather than silently adopting the
remote revision. Starting a different page selection clears the old editable
page/draft while awaiting its cache/network result. Five reducer regressions
cover edit/save/recovery, normal replacement and changed selection. Both
Direct/Store Wiki test suites and the module-size gate pass. No layout changes;
queued for the next batched APK, not yet published. Real-device slow-network
editing checks remain pending.

Wiki recovery persistence follow-up: background draft-save exceptions are
handled without escaping the ViewModel coroutine. Cancellation still
propagates. A matching active edit receives a warning without losing text;
failures for different selections/newer content are ignored. Encrypted
payload saves now return the preference commit result, and WikiDraftStore
checks it rather than silently claiming a failed commit succeeded. Four
regressions cover success, write failure, cancellation and stale failure
isolation. Direct/Store Wiki suites and module-size checks pass. Pending APK.
Remaining audit: serialize delayed draft saves and explicit clears so cancelled
IO cannot resurrect a cleared draft; avoid synchronous clear commits on UI.

Draft ordering follow-up: a shared WikiDraftWriteGate records a fresh intent
ticket before the debounce/IO dispatch. Older tickets cannot overwrite newer
edits or explicit clears. Storage reads/writes/clears serialize so a clear
follows a write already inside encryption/commit; the gate is shared across
store instances using the same encrypted preference file. Four tests cover
reverse execution, cleared-draft resurrection, concurrent in-flight storage
and lock release on failure. Direct/Store Wiki suites and module-size gate
pass. Queued for APK. Synchronous clear commits can still wait for storage;
moving those off the UI thread remains pending and is not claimed solved.

Async discard follow-up: explicit Wiki draft clears now record their intent
immediately and dispatch encrypted preference commits to a shared IO scope,
independent of screen destruction. Cleared drafts are hidden from recovery
reads immediately, including before the queued commit starts; delayed clears
cannot erase newer saves. Storage failures are logged without draft contents
and remain hidden for the current process. Three added tests cover dispatch,
read visibility and failure; both Android Wiki suites/module gate pass.
As with any asynchronous persistence, process termination or disk failure before
commit cannot guarantee the discard survives restart. No APK published yet.

Broader Android verification: all 579 tests pass per Direct and Store variant
(zero failures/errors/skips). Full lint remains running in the existing Gradle
invocation; it has advanced through Direct analysis and must be checked before
claiming lint success. No device is connected for physical checks. Follow-up
identified: recheck draft-read visibility after storage returns when a clear
was requested during that read.

Verification completion: Direct and Store full lint passed (combined Gradle
run 4m30s). Afterwards, recovery reads gained a completion-time intent check:
a clear or new edit arriving during decryption suppresses the superseded
result. Three regressions cover clear/new-edit interruption and unchanged
reads. Both Android Wiki suites/module-size gate pass after this small change;
the full lint result applies to the immediately preceding snapshot. No active
build job or new APK remains from this pass.

Offline Wiki encoding: moved encoding into WikiOfflineEncoding and replaced
repeated whole-snapshot serialization/drop-last retries with incremental
encoded-byte accounting. The encoder stops at the first non-fitting page,
retains the same newest-prefix eviction policy, and preserves JSON/cache
compatibility and the 3 MB limit. This avoids serializing all 128 large pages
on each trimming attempt. Added 128-page and Unicode/escaping boundary
regressions; Direct/Store Wiki suites and module-size gate pass. No quantified
device-speed or peak-memory claim yet; queued for the next APK.

Offline cache durability: payload/index commit failures now stop eviction of
older vault caches. Oversized uncacheable snapshots fail explicit offline
requests instead of silently succeeding. Automatic open/recent-page cache
updates use best-effort persistence with content-free logging, so storage
exceptions cannot break online reading or escape the background coroutine.
Three commit-order tests cover payload failure, index failure and successful
eviction ordering. Direct/Store Wiki suites and module-size gate pass; pending
APK. This is ordered persistence, not a transaction spanning preference files;
an index failure may leave the newly written payload unindexed.

Wiki offline network collection now retains a cacheable prefix plus one
four-page batch rather than all 128 fetched pages before trimming. Byte-budget
encoding/decoding runs off Main; downloading stops at the first non-fitting
batch, and existing partial-cache status reports remaining pages honestly.
Embedded-image discovery follows retained pages rather than dropped content.
Tests prove a 128-target/large-page case requests four pages and keeps two,
while small pages retain all targets in order with progress updates.
Direct/Store Wiki suites and module-size gate pass. This bounds page retention,
not total app memory: JSON, network buffers and image caching still add overhead.
Queued for APK; no device benchmark or new package yet.

Offline cancellation follow-up: optional image loading propagates cancellation
and checks coroutine activity before/after synchronous HTTP. Non-cancellation
image failures remain optional. Offline mirror callbacks now carry a vault-
visit token, separate from page selection, preventing old progress/results
from updating a reopened vault. Four asset-loader tests cover success, missing
images, explicit cancellation and cancellation during a blocking response.
Direct/Store Wiki suites and module-size gate pass. The underlying blocking
request is not forcibly aborted; its returned data is discarded on cancellation.
Queued for the next APK.

Offline snapshot ordering: merge now compares saved timestamps before page
deduplication and retains the newer tree/selection envelope; ties preserve the
existing incoming-wins behavior. Background current-page caching timestamps
the captured state before dispatch, not after waiting for IO. Shared locking
serializes payload/index reads and writes across store instances. Regressions
cover older late snapshots and equal timestamps. Direct/Store Wiki suites and
module-size checks pass; pending APK. Local timestamps are cache ordering hints,
not proof of server revision ordering or protection against wall-clock changes.

Android 0.16.72 / code 233 release preparation: version and current-release-only
update notes now describe the accumulated Wiki stability/offline-performance
batch. All 595 tests pass for each Direct/Store variant (none skipped), server
update metadata tests pass, and architecture/Store-readiness gates pass.
Full Direct/Store lint plus the signing guard remain in the existing Gradle
run. Do not deploy the updated server metadata before matching APK verification;
the live server/download remain 0.16.71. Optimized packaging has not started.

Release validation continuation: Direct lint passed for 0.16.72; Store lint and
the signing guard remain in the same live Gradle invocation. Preserved the
existing Direct APK at artifacts/releases/bobobooks-0.16.71-direct.apk; its
SHA-256 matches the live release (7d0387eb3f8afa1179a3b386dd97bb117f760bd9c5bdc3620364e82c5dee6abf)
and its certificate was reverified. Live service remains healthy, zero restarts.

0.16.72 validation is complete: both full lint tasks and the persistent-signing
guard passed (combined Gradle run 4m58s). Optimized Direct/Store assembly has
started in one Gradle invocation and remains running. Do not publish or deploy
the new update metadata until APK version/signature/readiness checks finish.

Release hygiene: excluded local .env variants, config/media directories,
compose.override.yaml, private Seafile config and Android local.properties
from Docker context, and added readiness checks enforcing those exclusions.
The archived release directory is Git-ignored. Full Go tests pass.
The first matching server build completed before this context fix; a fresh
build with exclusions is now running. Older local build-stage caches may still
contain private configuration; they were not purged, and no secrets were printed.
Final runtime images only copy the binary, empty data directory and entrypoint.
Keep old build caches private. Android optimized packaging is still running.

Android 0.16.72 / 233 Direct is now published (September 7, 15:23 UTC).
APK signature verifies against the existing certificate; version and package
readiness checks pass. Direct SHA-256:
559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6.
The live mounted download matches that hash. Matching server image
32533a4a68ea850525f5d4de57b2e5a089579f0e917a45b9ab4caaa767eefeae is
healthy with zero restarts and all ten libraries loaded. Direct APK archived
under artifacts/releases alongside 0.16.71. No media/config data was removed.
Store optimized assembly remains running in the existing Gradle invocation;
its final artifact is not yet verified. Real-device upgrade and Wiki/offline
behavior checks remain pending.

0.16.72 packaging completed: both optimized variants assembled successfully
(combined 7m57s), including R8 and vital lint. Store APK reports 0.16.72 / 233,
verifies against the persistent certificate and passes packaged readiness.
Its permission dump contains no installer or broad-storage permission.
Store SHA-256:
744e8cc7c639862400e4904cd1dd27d65c5c5e2385fe1191f922019b1903f9a4.
Archived at artifacts/releases/bobobooks-0.16.72-store.apk. Both packaged
variants pass readiness checks; no release build jobs remain. Live Direct
download/server remain healthy. This is a privately signed Store-flavor
verification artifact, not a Play Store submission or approval.
