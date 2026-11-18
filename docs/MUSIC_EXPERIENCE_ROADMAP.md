# Music experience roadmap

Mochil should feel like a first-class music app without turning its quiet,
media-neutral interface into a clone of another service. The design priorities
are fast playback, artwork-led discovery, readable metadata, predictable queue
control, and useful offline behavior. Music and audiobooks share transport
infrastructure but keep task-appropriate browsing and player controls.

## Current foundation

- Artists, albums, and tracks with cover or compact-list layouts.
- Search, favorites, downloads, offline filtering, ReplayGain, and gapless
  queue playback.
- Persistent synchronized queue and playlists.
- Background playback, notification/media-session controls, sleep timer,
  repeat, shuffle, speed, and a player that survives navigation.
- Phone, tablet/two-pane, and television presentation.

## Delivery sequence

### 1. Library hierarchy and track clarity — in progress

- Use one compact browse-mode menu instead of a permanent segmented strip.
- Give album and artist detail a clear artwork-led identity and primary Play and
  Shuffle actions.
- Within an album, show numbered, compact rows with artist and duration. Reuse
  one group cover instead of decoding the same artwork for every track.
- Keep favorites and downloads in the per-track action menu so titles retain
  maximum width.

List-mode albums, audiobook series, and audiobook authors now remain on the
root shelf and lazily expand one group at a time. Cover mode retains its focused
detail screen, and the Artist → Albums → Tracks hierarchy keeps its separate
drill-down rather than flattening artist releases.

### 2. Music home and discovery — in progress

- Add bounded sections for recently played, recently added, favorites,
  downloaded music, artists, albums, and playlists.
- Keep each rail short and provide an explicit destination for the full list.
- Restore the last meaningful music destination without forcing every app
  launch into the music library.

The bounded discovery surface is implemented for phone and TV. It loads short
recently played, recently added, favorite, album, artist, and offline sections; independent network
requests run concurrently and every rendered section has a strict item cap.
The web music shelf now uses the same bounded model for its Discover destination:
short recently played, recently added, favorite, album, and artist sections load
concurrently, with explicit full-list destinations and no full-catalogue request.
Its already-synchronized playlists now appear in a bounded eight-card discovery
rail using the same four-cover mosaics as the full playlist destination. Opening
one enters the focused playlist directly without another catalogue request.
Favorites and playlists also keep transport failure distinct from a genuinely
empty account collection. Discover continues rendering the sections that did
load while its quiet status reports the unavailable account data; the dedicated
Favorites and Playlists destinations no longer present misleading empty-state
instructions after a failed request.
On narrow touch screens, a deliberate horizontal swipe moves between the music
destinations in the same order as the compact rail. Horizontal discovery shelves
and interactive controls retain their own gestures, and short or mostly vertical
movement is ignored so ordinary scrolling remains predictable.
Recently played is account-owned, scoped to the selected shelf, and opens a
separate destination capped at twenty tracks on Android and web rather than loading full history.
The web Discover shortcut for Recently added now opens the complete Tracks
destination in Recent order instead of inheriting an unrelated alphabetical
sort from an earlier visit.
Favorites and Recently played honor their Cover/List preference: Cover mode
uses an artwork grid and List mode keeps compact rows, with the same contextual
track actions available in either layout.
On web, focused album details now keep their representative artwork, identity,
and primary Play/Shuffle actions together. The generic sticky actions disappear
from album roots and details, avoiding inactive or duplicated controls.
Album track lists use disc/track numbers instead of decoding the same album art
for every row. This makes ordering clearer and bounds artwork work to the single
representative cover already visible in the album header.

### 3. Now Playing and queue

- Refine the full player around artwork, title/artist, progress, primary
  transport, and a visibly reachable queue.
- Show the current queue item, preserve queue position, and make reorder/remove
  controls suitable for touch and remote input.
- Add clear Add next / Add to queue / Add to playlist actions from track and
  album menus, with optimistic feedback and conflict-safe persistence.

Track menus now provide Play next and Add to queue. Both update the live Media3
queue and the synchronized persistent queue. Track and whole-group insertions
show a bounded confirmation message, while playlist mutations report success
only after persistence completes.
The web track menu now has the same Play next behavior: it deduplicates the
chosen track, inserts it immediately after the currently playing item, and
persists the reordered queue without interrupting playback.
Track menus also open a compact playlist picker that can select an existing
playlist or create a named one. The server appends the reference atomically,
preventing concurrent add operations from replacing a playlist snapshot.
On touch devices, track rows and cover cards open this contextual menu on long
press instead of reserving permanent space for overflow icons. Normal tap still
starts playback, active downloads retain a compact progress indicator, and
accessibility services receive an explicit Track options action.
Playlists are now a first-class music browse destination. Their focused detail
view supports playback, ordered tracks, move/remove actions, atomic rename,
confirmed deletion, and artwork without exposing editing controls on every row.
On web, the root now shows compact artwork-led playlist cards instead of
expanding every track list at once. Opening a card enters a focused playlist
surface with one primary Play action; confirmed deletion remains inside its
options menu, away from ordinary browsing. Rename uses an inline form rather
than a browser prompt, while move-up, move-down, and remove actions reuse each
track's existing options menu and persist the resulting order atomically.
Album and audiobook-series menus can insert the whole ordered group next or at
the end of both the live Media3 queue and synchronized persistent queue.
Fine-pointer web clients can now drag queued tracks directly before or after
one another. The same serialized newest-state persistence used by the existing
move and remove actions prevents overlapping reorder requests from restoring an
older queue, while menu actions remain available to touch and keyboard users.
Expanded web Now Playing now exposes the live queue beside its primary
transport instead of making listeners leave the player for the catalogue
destination. The focused panel centers the active track, identifies it
semantically, and pages in bounded 60-row windows so a long synchronized queue
does not create an unbounded player DOM.

### 4. Collections and offline ownership

- Playlist artwork now uses a bounded four-cover mosaic with a representative
  cover fallback. Fine-pointer web clients can drag tracks before or after
  another track and persist the new order atomically. Move up/down actions
  remain available for keyboard, assistive-technology, and touch users.
- Add album and playlist downloads with bounded concurrency, progress, storage
  estimates, retry, and explicit removal.
- Distinguish available offline, downloading, failed, and stream-only states
  without persistent badges on every row.

Whole-album and audiobook-series downloads are implemented as a sequential
batch with completed/total progress, existing-download skipping, per-file
storage checks, and one final offline-index refresh. Playlist detail now reuses
that bounded pipeline for download and retry, summarizes known storage size and
offline count, and offers confirmed bulk removal without deleting the playlist.
Per-track state remains available through the existing track menu.

Web track menus now expose the authenticated download already available to
podcasts and audiobooks. A playlist can also be downloaded as one sequentially
streamed ZIP: source files are stored without wasteful audio recompression,
memory use stays constant, archives are capped at 1,000 tracks and 50 GB, and
only two archive streams may run server-wide. Focused album views now use the
same archive boundary through an opaque server-derived group identity. Albums
with the same title in different folders remain distinct, multi-artist tracks
in one album folder stay together, and disc subfolders resolve to their shared
album root. Neither the web nor Android clients need to trust a display-name
match.

Web Discover now adapts its compact track lanes to the available canvas: four
columns on wide desktop, three on intermediate desktop, two on narrow tablet,
and one on phones. This brings Albums and Artists into the useful desktop
viewport without shrinking their cover-led rails or changing mobile reading
order. Browser coverage verifies the wide composition rather than relying on a
stylesheet-only assertion.

### 5. Metadata and library quality

- Surface year, genre, disc/track number, album artist, and codec/quality when
  present; keep unknown fields out of the interface instead of emphasizing
  them.
- Add a focused metadata-quality report for unknown artists, missing artwork,
  duplicate releases, and inconsistent album tags.
- Preserve preferred-format and duplicate rules without merging distinct
  editions or releases.

Android now retains the server's existing album artist, genre, disc/track,
container, duration, and size fields and exposes them through a focused track
information dialog. Unknown fields stay absent from both the dialog and normal
track rows. The controlled audio-index schema upgrade now adds bounded release
year and bitrate values. Existing cache snapshots stay visible while unchanged
files are re-probed once under the new probe version; subsequent scans reuse the
upgraded cache normally. Migration is capped at 80 unchanged files per shelf
per scan so an upgrade cannot create a full-library FFprobe burst. Android shows
both fields only when known.

Web track menus now open the same quiet, artwork-led information surface with
artist, album artist, album, genre, year, disc/track, format, bitrate, duration,
and size shown only when known. Normal rows remain uncluttered. Album ordering
also reads the server's canonical `disc` and `track` fields, fixing the former
fallback numbering caused by web-only field names.

Web Metadata now offers a shelf-specific, read-only music quality check. It
reports unknown artists, exact duplicate track identities, conflicting album
tags within one folder, and albums without known embedded or bounded
folder-sidecar artwork. Attached-cover facts come from the existing probe cache;
the fast report never launches an unbounded media decode, and it only performs
one sidecar lookup when no track in an album is already known to contain art.
Quality grouping follows physical album folders (with CD/Disc subfolders folded
into their parent), so inconsistent tags cannot inflate the album total and
same tags in unrelated folders are not mislabeled as one duplicate release.

The audio probe cache now records whether a track contains attached artwork.
Album artwork fallback stays bounded, but it tries siblings already known to
contain a cover before unknown candidates. When an album genuinely has no
usable image, web rows render a deterministic, language-safe album monogram
instead of an empty square or generic music-note placeholder.

### 6. Television and large screens

- Use remote-first horizontal music rails, strong focus states, and a large
  Now Playing surface.
- Keep books hidden by default on TV while allowing users to override visible
  libraries.
- Verify queue, seek, shuffle, repeat, library switching, and background
  playback with D-pad and common game controllers.

Android TV now preserves the last meaningful music destination instead of
forcing Tracks whenever the shelf opens. Its compact global dock exposes each
transport action as an independent focus target and opens a dedicated,
remote-first Now Playing surface for artwork, seek, shuffle, repeat, and primary
transport controls. Returning from it restores the originating Home, library,
or Settings route rather than resetting navigation.
Now Playing progressively discloses a virtualized live queue instead of adding
another permanent shelf. The active item remains visible, Select starts any
queued item, and holding an item offers bounded move-earlier, move-later, and
remove actions. Starting music on TV now replaces both the live Media3 queue and
the account-persistent queue, matching mobile; subsequent music and audiobook
queue edits persist as well, while podcast episode queues remain session-owned.
Track cards now keep those surfaces visually quiet: holding Select opens Play
next, Add to queue, and favorite actions, with live and synchronized queues
updated together.

## Quality gates

- No unbounded catalogue or artwork loading; lists and rails remain paginated.
- Starting a track gives immediate visible feedback and never leaves two media
  sessions playing simultaneously.
- Playback, queue, and offline state survive navigation and process recreation.
- A paused or background player that loses a progress revision conflict yields
  to the newer device and cannot overwrite it during a later lifecycle save;
  explicit playback resumes ownership.
- Every icon-only action has an accessible name and at least a 48 dp target.
- Phone controls remain above gesture and three-button navigation insets.
- TV focus is always visible; e-ink avoids animation-heavy or translucent UI.
