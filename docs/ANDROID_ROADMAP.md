# Mochil Android roadmap

This is the delivery plan for turning the current Android client into a complete
mobile application. Android phone support is the active product. Android TV is
a later presentation layer over the same data and playback foundations. Kobo is
a separate compatibility track and must not delay a dependable phone app.

The app is also expanding into a standalone client: local SAF folders, direct
podcast feeds, OPDS, and read-only WebDAV sources must work without a Mochil
server. The source-adapter boundary and staged delivery are specified in
[`ANDROID_STANDALONE_SOURCES.md`](ANDROID_STANDALONE_SOURCES.md).

## Product rule: browse media as collections, not one long list

Every hierarchical library uses the same predictable interaction:

| Library | Browsing hierarchy |
| --- | --- |
| Books, manga, comics | Series → volumes |
| Music | Artists → albums → tracks |
| Audiobooks | Series → books/parts → chapters |
| Podcasts | Shows → episodes |
| Television | Shows → seasons → episodes |
| Movies and documents | Flat collections with filters |

- **Cover view:** show the parent collection as a grid and open a dedicated
  detail screen when selected.
- **List view:** show compact parent rows, collapsed initially. Selecting a row
  expands it in place and lazily fetches only that collection's children.
- Remember cover/list choice per media type and browse level.
- Preserve the selected library, browse mode, filters, scroll position, and
  expanded group while navigating away and back.
- Search and sorting apply to the current level. Large lists use cursor paging,
  automatic continuation, and an optional `#`/A–Z jump rail.
- The server owns grouping, ordering, preferred-format deduplication, and
  pagination. The phone must not download an entire catalogue to regroup it.

## Definition of mobile v1

Mobile v1 is ready when a person can sign in, find media through its natural
hierarchy, inspect it, stream or download it, resume it, and manage offline
content without falling back to the website. It must remain usable on a slow
connection, after process recreation, and with either gesture or three-button
system navigation.

The first supported scope is:

- Books, manga, and comics: series/all/authors/recent browsing, details,
  download, open, and synchronized reading progress.
- Music: artists/albums/tracks, queues, playlists, favourites, background and
  offline playback.
- Audiobooks and podcasts: grouped browsing, chapters/episodes, resume, speed,
  sleep timer, downloads, and played state.
- A clean phone shell with Home, Library, Downloads, and Settings, plus a
  persistent mini-player that never overlaps system controls.
- Clear loading, empty, offline, expired-session, permission, and retry states.

Video can follow this quality bar as a mobile v1.1 feature. Executable extension
runtimes are outside the product scope; interoperable sources must use bounded
first-party or declarative adapters.

## Delivery sequence

### M0 — stabilize the app shell

- Replace the growing row of library chips with an app-bar library selector.
- Add real navigation and routes with system Back and predictive-back support.
- Use bottom navigation for Home, Library, Downloads, and Settings; adapt to a
  navigation rail on wide screens.
- Keep the mini-player immediately above app and system navigation, including
  Samsung three-button mode, gesture mode, landscape, cutouts, and the keyboard.
- Restore route, selected shelf, scroll position, and player state after
  configuration changes and process recreation.
- Establish reusable loading, empty, error, retry, and offline components.

**Exit gate:** sign-in, shelf switching, background music, notification controls,
Back navigation, rotation, and both Android navigation modes work without lost
state, overlap, or playback interruption.

The native Home dashboard now exposes authenticated Continue reading and
Continue listening activity, with direct resume actions and compact
Home/Library navigation.

### M1 — books and series: complete the missing basic experience

- Add browse choices: Series, All books, Authors, and Recently added.
- Make Series the default for publication libraries when series exist.
- Add independent Cover/List preferences for series and books.
- Implement collapsed series rows with lazy volume loading in List view.
- Implement a series detail screen in Cover view, ordered by series index with
  unnumbered extras kept separate from the main reading order.
- Add author detail and book detail screens with cover, title, author, series
  position, format, size, description, and available actions.
- Respect the server's preferred-format result so EPUB/PDF editions do not look
  like accidental duplicates; expose alternate formats from book details when
  useful.
- Add search, sort, filters, cursor infinite scrolling, and `#`/A–Z jumping.
- Add deterministic placeholder covers and useful empty/error copy.

**Exit gate:** a large library can be browsed from Series → Volume without a
flat wall of books, List groups can be collapsed/expanded, and returning from a
book restores the exact browsing position.

This is the first hands-on feedback checkpoint.

Recently Added is now active on mobile and TV. Ordering and keyset continuation
remain server-owned; the versioned publication index keeps warm deep pages from
sorting or scanning the complete shelf. Author grouping is now active on mobile
and TV with cover/list roots, search, alphabet filtering, cursor continuation,
and an author detail view backed by exact server-side book filtering. Series
and author details own an independent cursor and continue automatically on
mobile and TV, so opening a large group never truncates it or corrupts the
surrounding shelf position.
Mobile publication and music roots now use one compact browse-mode icon and
checked menu instead of permanently occupying the content width with segmented
labels. Search, filters, and cover/list controls remain separate recognizable
actions.
Publication cards keep the content-first interaction consistent: tapping opens
the publication, while long-press opens one options sheet for Book details and
collection membership. Details no longer depend on an unused callback threaded
through every grid, list, author, series, and collection renderer, and no
redundant trailing action icon is required.
Music albums plus audiobook series and authors now implement the shared List
rule: rows start collapsed, open one lazily fetched track group inline, and
collapse without leaving the shelf. Cover mode retains dedicated details.

### M2 — downloads and publication opening

Interrupted app-private publication, music, audiobook, and podcast downloads now
retain a partial only when the server supplied a strong ETag or Last-Modified
validator. A retry uses `Range` plus `If-Range`, appends only after an exact
`206 Content-Range` match, restarts when the server returns a complete body, and
rejects mismatched ranges rather than mixing file versions. Startup cleanup
preserves these validated partials and removes orphaned or malformed state.
Atomic final publication and the existing byte/free-space bounds remain in
force. Moving explicit user-queued transfers into WorkManager with foreground
progress, pause, and cancellation remains a later UI/service slice; opening a
publication continues to prepare it immediately.

- Store downloads in app-private storage and index them locally.
- Use resumable WorkManager jobs with progress notifications, retries, checksum
  or size validation, and safe cleanup of partial files.
- Add Wi-Fi-only, charging, storage-limit, and remove-after-reading preferences.
- Provide a Downloads screen grouped by downloading, available offline, and
  failed, with pause/cancel/retry/remove actions.
- Open EPUB, PDF, CBZ/CBR, and supported documents in an installed compatible
  reader until each native reader is ready.
- Never modify the canonical server or storage file.

**Exit gate:** interrupted downloads recover cleanly, offline files open without
network access, and clearing one download cannot remove another item.

### M3 — native reading and accurate progress

**Current status:** Native CBZ and EPUB rendering are implemented. EPUB uses
Readium structural locators and text anchors and includes contents, bounded
search, bookmarks, typography sizing, themes, and paged/scroll presentation.
Android EPUB highlights are now created through native text selection, rendered
with theme-aware Readium decorations, stored offline in bounded atomic files,
and synchronized incrementally with tombstones and explicit same-item conflict
choices. The authenticated server also provides a deterministic account-only
JSON export and an exact-confirmation deletion endpoint for the server copy.
Highlights now support inline add/edit/remove notes with UTF-8 byte bounds,
offline persistence, revision-safe synchronization, IME-aware editing, and
conflict gating. KOReader now captures and uploads highlights with stable IDs,
native locators, text anchors, revisions, and deletion tombstones; Android
resolves those text anchors into Readium decorations. Android Settings exposes
streamed account export and exact-confirmation deletion of the server copy.
Android-origin highlights import into KOReader with stable cross-device IDs.
A shared versioned fixture corpus now validates Android and server decoding,
text anchors, Readium locator payloads, KOReader XPointers, and lossless wire
round trips. Bounded structural XPointer conversion now maps KOReader
DocFragments and element paths to Readium spine/CSS locators in both directions;
character-offset conversion across arbitrary EPUB DOM normalization remains in
this milestone. Native
PDF paging, structural page synchronization, explicit cross-device conflict
choices, local page restoration, bounded neighboring-page rendering, and
external-reader handoff are implemented. Fixed-layout save acknowledgements
require an exact page match on both server and Android; nearby normalized
fractions cannot silently accept an adjacent stale PDF or comic page.

- Ship an EPUB reader with typography, themes, table of contents, search,
  bookmarks, and highlights.
- Add paged comic/manga reading with left-to-right/right-to-left direction,
  fit modes, and prefetch bounded to nearby pages.
- [x] Add native PDF paging after EPUB and comics are stable, with bounded
  rendering, structural page synchronization, explicit conflict handling,
  local restoration, and external-reader handoff.
- Synchronize structural positions rather than display percentages. Resolve
  concurrent device updates explicitly and retain a local unsynced state while
  offline.
- Add a shared annotation layer after publication identity and position sync
  are stable: text-anchored highlights, notes, bookmarks, offline conflict
  handling, KOReader import/export, and explicit export into Mochil Notes.
  The server foundation and Android highlight client are implemented as an isolated account/publication
  store with per-item revisions, bounded incremental batches, persistent
  deletion tombstones, conflict-preserving APIs, account-scoped export, and
  confirmed deletion of the server copy. Android capture, merge UI, and native
  locator application and editable highlight notes are active. The server can
  now create a bounded Mochil Notes page from one publication's live
  annotations through an explicit one-way adapter. Bidirectional KOReader
  transfer and shared KOReader/Readium contract fixtures are active. The
  Highlights tab can explicitly synchronize the current publication and create
  a bounded Mochil Notes page, with conflicts and offline failures left
  recoverable in place.
- Offer external-reader handoff even when a native reader exists.

**Exit gate:** reopening on the same or another device returns to the correct
logical location and offline progress synchronizes without silently replacing a
newer position.

### Native wiki and vaults

Android now recognizes server libraries of type `vault` and opens a native,
virtualized folder tree instead of the generic unsupported-library state. The
client reads the server's Obsidian-compatible hierarchy, excludes hidden vault
internals through the shared server index, renders bounded Markdown headings,
paragraphs, lists, task lists, properties, images, quotes, and fenced code with
copy support, and allows writable vaults to create folders/pages or edit existing
pages. Saves use the server edit revision and surface conflicts rather than
overwriting changes from Obsidian or another client. Phone navigation uses tree
then page; wider Android surfaces use a fixed tree and reading pane. Wiki links,
backlinks, rename/move, recoverable vault deletion, debounced full-text search,
one bounded encrypted active draft, and revision-checked interactive Markdown
task toggles are implemented. Task toggles change only the selected checkbox
marker, preserving frontmatter, line endings, and all unrelated vault text.
The compact scrollable editor strip now covers headings, bold, italic,
strikethrough, inline and fenced code, bullet and numbered lists, tasks,
quotes, web links, and Wiki links. Block actions operate on complete selected
lines and emit ordinary Obsidian-compatible Markdown; fenced-code placeholders
remain selected for immediate typing.
Task updates render optimistically and roll back exactly when the write fails.
The encrypted draft now survives process death through the automatic initial
page selection: only a draft matching that exact vault and page is retained
and recovered. Ordinary navigation, confirmed discard, cancel, close, and save
still invalidate the old draft, and normal page changes avoid encrypted-store
reads entirely.
Obsidian-style Markdown tables render natively with inline Wiki links and
bounded row, column, and cell sizes; wide tables scroll inside the page rather
than forcing the whole reader surface wider.
Page Markdown and its outline now come from one cancellable parse on the
background dispatcher instead of two parses on the Compose main thread. Large
vault pages show a quiet accessible formatting state, and heading navigation
waits for both the network page and parsed document before consuming its jump.
Obsidian heading links now resolve both across pages and within the current
page: `[[Plan#Details]]` waits for Plan to finish loading before jumping to the
matching native outline block, while `[[#Details]]` moves within the open page.
The pending anchor is consumed once so a later refresh cannot unexpectedly
move the reader again.
The four most recently saved vaults each retain their own tree and up to 128
recently viewed or explicitly mirrored Markdown pages in an encrypted,
three-megabyte-bounded snapshot. Opaque per-vault storage
identities keep library names out of preferences and keystore aliases, while
the former single-vault snapshot migrates on first use. Reopening can paint the
selected page before the network round trip, and recent cached pages remain
browsable during an outage; saved copies are deliberately read-only until the
server confirms each current revision. Old one-page snapshots migrate in
place. Cache load/save diagnostics record only duration and outcome, never
vault names, paths, or content.
An explicit **Save vault pages offline** action lives in the existing page menu.
It fetches Markdown with four bounded workers, reports live page progress, and
states exactly how many pages fit if the 128-page or three-megabyte cache limit
is reached. Referenced images are mirrored separately into no-backup app-private
storage with 32-image, three-megabyte-per-image, and 24-megabyte-total bounds.
Partial image coverage is reported honestly, same-revision images survive a
transient partial refresh, and changed server revisions invalidate stale files.
Cached pages retain their native outline while read-only. If server full-text
search is unavailable, the tree keeps useful local page-name/path filtering
instead of replacing cached matches with an empty result.
Every asynchronous vault mutation is owned by its initiating library and page,
so a late save, create, move, or trash response cannot replace a page selected
while that request was in flight. A bounded native outline is progressively
disclosed through the existing page menu; it preserves Markdown heading depth,
jumps to the exact rendered block, and keeps scroll state isolated per page.
Revision history remains
later native-wiki work; external Obsidian vault files remain the canonical source.

Exact Readium page locators are now checkpointed locally on every page change
and synchronously when the reader backgrounds. Renderer-neutral structural
locators remain separate for KOReader synchronization. Offline checkpoints are
marked pending and retried on reopen even without another page turn, so local
restoration never depends on network availability. The transient reader chrome
shows the real checkpoint state (syncing, server-acknowledged with time, or
saved on device) without adding permanent reading clutter.
Remote revision ownership, initial cross-device position resolution, conflict
blocking, and debounced save outcomes now live in a gateway-backed EPUB progress
coordinator rather than the reader Activity. The Activity retains presentation,
Readium navigation, and lifecycle checkpoint responsibilities.
Local annotation mutation, tombstone persistence, conflict resolution, anchor
repair, pull/apply synchronization, and atomic Notes export now live in a
separate annotation controller. Android selection mode and Readium decoration
submission remain presentation adapters in the Activity.

### M4 — complete audio libraries

- [x] Change Artists from Artist → all tracks to Artist → Albums → Tracks.
- [x] Allow collapsed album/artist rows in List view and detail screens in Cover
  view, using the shared grouping rule.
- [x] Add album/artist play and shuffle, current queue, reorder/remove, favourites,
  playlists, seek, repeat, and ReplayGain preferences.
- [x] Add app-private offline music playback and downloaded-only filtering,
  retaining enough local metadata and the last known shelf list to work without
  a server connection.
- [x] Add audiobook Series/Authors/All views, chapters, Continue listening, speed,
  sleep timer, bookmarks, and offline playback.
- [x] Add native podcast Shows → Episodes browsing, cover/list layouts, search,
  played/unplayed filtering, cursor continuation, resume, and app-private
  offline episode playback.
- [x] Add direct RSS subscription and confirmed unsubscribe controls.
- [x] Add bounded directory discovery with authenticated artwork and manual
  per-show refresh.
- [x] Add explicit shared server-cache auto-download and retention policy
  controls, clearly distinguished from device-local downloads.
- [x] Add per-show device-local auto-download, retention, unmetered-network
  policies, downloaded filtering, and offline show fallback.
- [x] Preserve a bounded, credential-free audio download index in a selected
  user folder and recover its music, podcast, and audiobook files when that
  folder is selected after reinstall. Never guess legacy ownership from a
  filename alone. Recovery also runs for an already-selected folder at app
  startup, and the active Music and Podcast views refresh their offline state
  as soon as recovery completes instead of requiring another app restart. The
  recovery planner admits only readable indexed documents, deduplicates a
  damaged sidecar by stable media identity, and never replaces a current
  app-private record. Removing a download is transactional across the private
  and selected-folder copies: a failed external deletion remains indexed and
  playable instead of being falsely reported as removed, while removing the
  final download writes an empty sidecar so reinstall recovery cannot resurrect
  stale entries.

Podcast subscription and episode identities are deterministic for the same
feed address and stable GUID/enclosure. Removing and later re-adding that feed
therefore reconnects its sidecar-indexed device downloads instead of creating
unrelated episode identities; a server regression test protects this lifecycle.
- [x] Add an explicit Queue view shared by audio libraries; keep downloads as
  contextual filters and Settings inventory rather than a permanent bottom tab.
- Keep playback alive across navigation, screen lock, Bluetooth changes, and
  normal process pressure; expose correct artwork and metadata to system media
  controls.

The Media3 service now persists a bounded active timeline, current item,
position, speed, repeat, and shuffle state every 15 seconds and durably at
normal shutdown. It restores that state paused after process recreation,
rebuilds authenticated stream routes from the device's current server address,
and retains only the current item and neighboring chapter maps to bound storage.
Device interruption testing remains part of the M6 release matrix.
Podcast launches on mobile and TV now refresh the selected episode's exact
account progress immediately before playback. This avoids stale resume points
when another device advanced the episode after the current shelf was loaded,
while retaining the cached shelf position as an offline fallback.
That server-derived start position carries a bounded, one-shot seek marker into
Media3. The service consumes only the matching initial seek as passive; later
scrubs remain explicit and may deliberately claim cross-device ownership.
Media3 queue transitions also checkpoint the outgoing podcast or audiobook
before the current item changes. Queue metadata carries the bounded duration,
so manual skips retain their exact position and natural completion persists
without probing the file again.
Mobile playback presentation now lives behind `MobilePlaybackHost`: the mini
player and expanded player are the only composables observing the one-second
playback clock. Dashboard, library, source, and settings composition no longer
restart for elapsed-time updates. Settings observes a distinct ReplayGain flow,
so changing that preference remains immediate without inheriting clock ticks.
The mobile application shell now owns one typed root destination instead of a
string route plus a separate Settings flag. Android's system Back gesture
returns Library, Saved, Sources, and Settings to Home, while later-composed
media drill-down handlers retain priority inside their own surfaces.

**Exit gate:** every audio type can be discovered by its natural grouping,
queued, controlled in the background, resumed, and played offline.

### M5 — mobile video

- [x] Add Movies and Shows sections with Cover/List views and compact search.
- [x] Use Shows → Seasons → Episodes grouping.
- [x] Support direct play, server-provided HLS fallback, subtitles, resume, and
  Continue Watching.
- [x] Add in-player audio-track selection and picture-in-picture.
- [x] Clearly explain unsupported codecs instead of failing with a generic error.
  Native playback distinguishes unsupported decoding, malformed media, network
  loss, server delivery, and unavailable local files. Each state provides a
  short recovery action and records only a privacy-safe diagnostic category.

### M6 — mobile release quality

- Add tablet/two-pane layouts without stretching phone screens.
  Publication shelves now keep browsing beside series and book details at the
  840 dp expanded-width breakpoint. Music, audiobooks, podcasts, and video use
  that same breakpoint to keep their grouped browse pane beside the selected
  album, artist, series, podcast show, season, or episode list.
- Complete TalkBack labels, logical focus order, 48 dp targets, scalable text,
  contrast, reduced motion, and keyboard/switch access.
  Browse cards and list rows now expose explicit action labels and button roles
  across Home, publications, downloads, music, playlists, podcasts, video, the
  mini-player, and EPUB reader chrome. Device TalkBack traversal, enlarged-text,
  keyboard, and switch-access verification remains in the release matrix.
- [x] Add persistent light/dark/system themes with Mochil, E-ink,
  Manjaro-inspired, Ubuntu-inspired, and Material-inspired palettes.
- Add signed release builds, update checks, crash-safe diagnostics export, and a
  clear version/support screen.
  Update checks and bounded diagnostics are active. Release tasks now require
  an owner-controlled keystore and refuse unsigned production output; the first
  signed artifact still requires the owner signing credentials.
- Test large catalogues, poor networks, offline startup, low storage, expired
  tokens, server upgrades, process death, and media interruption.
  Expired catalogue sessions now clear the revoked encrypted token, preserve the
  server address for reauthentication, and recreate all session-bound models
  after login. Android backup and device-transfer rules explicitly exclude all
  private application state. Offline startup now falls back to the cached shelf
  list, automatically opens the Library when local media is the usable path,
  scopes publication and audio views to completed downloads, and exposes one
  deliberate reconnect action without retry loops. Failed continuation pages
  retain already-loaded music. Upgrade and process/interruption evidence remains
  in the release matrix. Offline book,
  music, audiobook, and podcast transfers now share bounded size and free-space
  enforcement, retain a 64 MiB operating reserve, publish only atomically
  completed files, and remove abandoned partials at the next process start.
  Playback snapshots are additionally committed when playback pauses and updated
  on playback start, queue transitions, and explicit seeks, reducing the state
  window exposed to abrupt process removal. Media3 owns audio focus and noisy-route
  handling through the foreground media session.

## Android TV — after the mobile release gate

TV reuses authentication, models, repositories, caching, downloads where
appropriate, and Media3 playback, but gets a separate Compose TV presentation.
The phone interface must not simply be enlarged.

Across TV, mobile, and web, long selectors use a shared compact-viewport rule
instead of permanently occupying content space. TV renders focus-driven,
virtualized rails whose choices scroll beneath a stable D-pad focus area;
mobile uses touch-sized compact menus or swipeable rails; web uses compact
sticky selectors with keyboard-aware virtualization. The information hierarchy
stays consistent, while each client retains its native input model.

**Current foundation:** The shared APK is discoverable by Google TV and other
Leanback launchers without declaring a touchscreen requirement. Runtime TV
detection selects separate ten-foot Home and library surfaces with large D-pad
focus treatment, five-item Continue Watching/Listening/Reading rows, current-audio
control, virtualized cover grids, and series/show/group drill-down. Authentication,
account activity, artwork, video launch, reading handoff, and Media3 playback reuse
the mobile data layer. Dedicated TV Settings provides update checks/install and
connection management. TV library search, podcast browsing/playback, episode
sorting/filtering, and focused-item metadata are implemented; richer dedicated
details and remaining device preferences remain separate presentation work. Book, music, audiobook, and
podcast grids request their next bounded cursor page shortly before remote focus
reaches the end, retain virtualization, suppress duplicate requests, and show a
restrained loading tile instead of requiring a disruptive Load more action.
Root TV shelves also expose compact, D-pad-focused browse modes: Series/All for
publications, Albums/Artists/Tracks for music, and Series/Authors/Titles for
audiobooks. These controls leave the surface during drill-down so the current
series, artist, or album remains the only navigation context.
Focused TV covers now populate a fixed-height detail area with title, media-
appropriate metadata, resume timing, and bounded descriptions where available,
without shifting the grid or opening another screen. Native focus-restorer
groups retain the last D-pad child independently in the library strip, browse
selector, and virtualized catalogue.
Virtualized catalogue positions are isolated by shelf, browse mode, hierarchy,
layout, search, alphabet bucket, and podcast sort/filter state. Returning to
one view cannot inherit an unrelated view's offset, and the access-ordered
position history stays bounded to 96 routes during long TV sessions. Each
route now retains its focused item identity alongside the scroll offset, so
returning from a series, show, artist, or album restores the exact card when it
still exists. Removed or no-longer-loaded items fall back to the stable edge
navigation instead of leaving focus detached. That recovery also observes
in-place refresh and filtering results, so a card disappearing without a
hierarchy transition cannot leave the remote on a dead focus target; ordinary
paging keeps the current card and scroll position intact.
Media shape now follows content instead of one oversized universal card:
publication and show roots retain covers, albums use square artwork, episodes
use compact landscape treatment, and track/podcast episode shelves use dense
wide rows. Focus-detail state is isolated from the grid so moving the remote
does not invalidate the virtualized catalogue. Show drill-down reuses the
compact selector slot for seasons, and Settings keeps only three focus-driven
categories visible at once while the rail scrolls beneath focus.
Theme, brightness, subtitle-preset, and streaming-quality choices in TV
Settings also use bounded lazy option rails. D-pad focus brings off-screen
choices into view instead of allowing fixed chip rows to clip at 720p or under
enlarged system text.

TV Home exposes every configured library in a large horizontally scrollable
focus strip. Remotes use ordinary D-pad navigation and select; PlayStation,
Xbox, and compatible Android controllers may additionally cycle libraries with
L1/L2 and R1/R2. Media buttons remain dedicated to playback.
The Home selector keeps libraries and Settings in one focus row rather than
spending a second line on a redundant “Libraries” heading, leaving more of a
720p viewport available for continuation content.
The TV composition root no longer subscribes to library state it never renders;
route-level collectors remain the catalogue observers, avoiding whole-app
recomposition as bounded pages arrive.
The default Espresso/Oat palette now uses the same softened background and
surface family on Android and web. Oat's deeper terracotta primary preserves
the warm character while meeting normal-text contrast with its button ink.

Held Select/OK/Enter and controller-A input is handled explicitly instead of
depending on a touch gesture abstraction. Continue Watching uses that contextual
menu without a permanent close button, and episode cards offer watched/unwatched
actions while updating progress in place.
Mobile video cover and list layouts use the same quiet interaction: tap plays,
while long press opens watched/unwatched actions without adding card chrome.
Podcast episode rows reuse the remote hold menu for played/unplayed state and
disable the action while its account-scoped update is in flight.
The compact drill-down selector exposes newest, oldest, and title ordering plus
all, unplayed, played, and downloaded filters. Held-card popups consume the
opening remote-button release, so opening or canceling a menu cannot trigger its
first action.
Continue Listening uses the same held-card removal path. Before playback, TV
re-resolves the selected item against the account progress endpoint and copies
the synchronized position into the playable Media3 track, preventing a stale
Home snapshot from restarting a podcast.
Continue Watching and Continue Listening artwork now includes a restrained
four-pixel progress line, providing resume context without adding another card
label or control. When audio is active, TV reserves space for the playback dock
on Home, library, and Settings routes so the final content row remains reachable
and visible. That reservation responds only when the dock appears or closes;
the playback clock remains isolated inside the dock and does not recompose the
catalogue every second.
Music track cards keep tap-to-play and expose Play next, Add to queue, and
favorite changes only through the guarded held-card menu. Queue actions update
both the live Media3 queue and the account-persistent queue.
The full TV transport follows media intent rather than exposing one universal
button row: music retains Shuffle, while podcasts and audiobooks use that same
bounded D-pad slot for the current playback-speed control. The speed label is
locale-stable and changes immediately as the shared Media3 rate cycles.

The shared reader input policy accepts hardware Page Up/Page Down, remote D-pad
Left/Right, and controller L1/R1 in EPUB, PDF, paged comics, and webtoon mode. An opt-in Reading setting maps volume keys to
previous/next page for BOOX and similar e-ink devices that expose page buttons
as volume controls; ordinary phones retain volume behavior by default.

The EPUB reader now learns pace from active forward-reading samples and shows
chapter/book time remaining only after enough samples exist. A Stats tab keeps
today and total reading time plus average page/position duration; idle gaps over
two minutes are excluded. Reader defaults include publisher typography and
built-in serif, sans-serif, monospace, OpenDyslexic, and accessible faces with
a live size preview. Users can import bounded TTF/OTF files through Android's
document picker; fonts are validated, copied to private offline storage,
previewed in Settings, and injected into reflowable EPUB resources. The same
preference model remains reserved for Kobo clients.

- Ten-foot typography, overscan-safe spacing, clear selected focus, and complete
  D-pad operation.
- Rows of covers for Home/Continue, Libraries, Movies, Shows, Music, and audio.
- Show → Season → Episode and Artist → Album → Track detail flows.
- Lean-back video and audio controls, subtitle/audio selection, and next-item
  behavior.
  Video and anime episodes now enter a bounded, validated Media3 queue from the
  current season. Standard next controls remain transient with the player,
  automatic transitions mark the completed episode, and manual skips preserve
  the prior position. Standalone movies remain single-item playback.
- [x] Pairing/code sign-in so credentials do not need to be typed with a
  remote. The TV submits only its server address, displays a six-character
  short-lived code, and polls with an unguessable request secret. Approval
  requires an authenticated browser session and issues the existing bounded,
  revocable device token exactly once; the account password never reaches the
  television.
- TV Settings now includes a bounded readiness check that observes D-pad,
  Select, and media-button input without intercepting navigation, reports HDMI
  output and Android's app-memory class, and states the playback display policy.
  Video explicitly keeps the display awake while audio allows the system screen
  saver. Final release evidence still requires running the check on the target
  physical remote/HDMI-CEC combinations and observing long-session memory.
- TV focus continuity now treats the catalogue as the user's active workspace:
  controller shoulder-button library changes preserve spatial focus instead of
  forcing it back to the edge rail, closing search restores the search trigger,
  and compact cards share one distance-legible focus scale and outline. Generic
  remote instructions are omitted from the header so focused metadata and media
  remain the visual priority.
- Returning from Settings also resolves the remembered Home library against the
  currently visible library strip. If that library was hidden or removed while
  Settings was open, focus falls back to the first visible library (or Settings
  when none remain) instead of targeting a detached card and leaving D-pad
  navigation unresponsive.
- Library request failures retain that remote-first contract: the error owns a
  directly focused Try again action, and retrying reloads only the failed group
  or shelf when possible instead of discarding the surrounding catalogue.
  Home failures use the same explicit recovery on TV and mobile; pull-to-refresh
  remains available on touch devices but is no longer the only discoverable way
  to recover.
- If playback is cleared while the full TV player is open, its empty state keeps
  the remote usable with an immediately focused Back action instead of leaving
  a screen with no selectable target.

## Kobo compatibility — discovery track, not an Android milestone

Do not assume the Android APK can become a Kobo app. Keep Kobo support as three
increasingly expensive options:

1. Maintain excellent OPDS support and the existing KOReader integration. This
   remains the dependable Kobo path.
2. Explore a lightweight Kobo-browser experience for catalogue, download, and
   progress synchronization if the device browser proves capable enough.
3. Consider a dedicated Kobo/Nickel application only after a prototype proves
   installation, networking, rendering, updating, and recovery can be supported
   without fragile device-specific maintenance.

All platforms should share stable server APIs and progress semantics; only the
presentation and device integration should differ.

## Architecture guardrails

Keep the application feature-oriented so completing the roadmap does not create
god files:

```text
app/                 navigation, dependency assembly, top-level shell
core/auth/           credentials and token lifecycle
core/network/        authenticated transport and API errors
core/model/          shared wire/domain models only
core/database/       local catalogue, downloads, and progress state
feature/books/       browse, series, details
feature/reader/      EPUB, comics, PDF adapters
feature/music/       artists, albums, tracks, playlists
feature/audiobooks/  series, chapters, listening state
feature/podcasts/    shows, episodes, subscriptions
feature/video/       movies, shows, seasons, playback
feature/downloads/   transfer orchestration and offline inventory
playback/            Media3 service, queue, session, notifications
```

- One route-level state holder per feature screen; no application-wide catalogue
  ViewModel and no UI file that owns networking, persistence, and rendering.
- Repositories expose paging streams and small domain operations. Compose screens
  render immutable state and emit user actions.
- Mobile and TV have separate UI modules while sharing core and feature-domain
  modules.
- Introduce a shared abstraction only after at least two real features need it.
- Every new endpoint is bounded, authenticated, cancellable, and covered by
  contract tests.

## Verification matrix

Each milestone includes unit tests for ordering/state reducers, API contract
tests, Compose navigation/state tests, and at least one release-build smoke test.
Before calling mobile v1 complete, run the full journey on:

- a small phone and a large Samsung phone;
- gesture navigation and three-button navigation;
- portrait and landscape;
- light/dark theme and enlarged system text;
- Wi-Fi, throttled network, offline, expired authentication, and low storage;
- a small library and a catalogue large enough to exercise several cursor pages.

## Immediate implementation order

Current foundation: the compact app shell, durable app-private publication
downloads, full Media3 player sheet, native paged/webtoon CBZ reader, Readium
EPUB reader, structural progress, and bidirectional Android/KOReader EPUB
highlight synchronization are implemented. EPUB device verification remains
an active release gate.

1. Complete audiobook chapters, bookmarks, and offline metadata.
2. Run the mobile accessibility and poor-network release matrix.
3. Begin mobile video only after the mobile v1 audio and reader gates pass.
