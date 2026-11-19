# Mochil design

## Product boundary

Mochil is a read-only catalogue, browser reader, and acquisition gateway over
libraries that already exist. It does not organize, edit, convert, or download
metadata for books. Canonical data remains in EPUB/CBZ files, optional
manifests, and the configured storage system.

## Interface direction

The web interface is a warm modern cinema library: near-black espresso or soft
oat backgrounds, restrained apricot controls, editorial typography, and cover
art as the primary material. A slim, borderless masthead remains available
while scrolling and condenses to the B mark on narrow screens. The active shelf
selector forms a second compact sticky tier, so navigation stays close without
turning into a large floating panel. The canvas
stays edge-to-edge and separates catalogue organization from presentation.
Readers browse by Series or All Books, while independently choosing list or
cover layouts for the series index and for books. Series Covers use the
earliest numbered volume as representative art and drill into the reader's
remembered book layout.
OPDS-compatible `#` and A–Z buckets remain swipeable and sticky. Light/dark mode
follows the device initially and remembers an explicit choice. The reader stays
edge-to-edge and visually separate from library chrome.

The server has four responsibilities:

1. Normalize embedded, manifest, folder, and filename metadata.
2. Build fast disposable indexes for navigation and search.
3. Publish compatible OPDS 1.2 feeds and optional OPDS-PSE links.
4. Deliver local files with HTTP range support.

Ease of installation is a product requirement. A user must not need to know
Go, Nix, OPDS XML, JSON, or systemd to get a working catalog.

## Shipped visual system

The library shell uses two warm themes. Espresso is the default dark canvas
(`#171513`) with parchment text (`#f4eee6`) and apricot selection
(`#f09a72`). Oat is the light canvas (`#f5efe6`) with espresso text
(`#241f1a`) and a deeper burnt-apricot selection (`#c85d3a`). Covers are the
main source of saturated color; surfaces, rules, and controls stay within the
warm neutral range. The destructive action is the only separate semantic red.

Newsreader is the editorial face for the wordmark, page headings, and empty
states. Source Sans is the compact humanist face for catalogue cover and list
titles, controls, metadata, forms, and status copy. Cover titles use a restrained
14–15px semibold scale so long names remain useful on phones. Both are self-hosted WOFF2 assets,
so the interface does not depend on a third-party font service.

The catalogue is a full-width cover canvas with fluid side padding rather than
a centered content card. Cover grids auto-fill from 138px columns, increasing
to 170px on very wide screens; phones use two equal columns. Book covers use a
2:3 frame and series representatives use 3:4. Titles clamp to two lines and
missing or failed cover images become deterministic typographic covers derived
from the title, preserving browsing when artwork is unavailable.

The borderless masthead contains the wordmark, Search, Library, Settings, and
theme controls. Selecting the wordmark returns to the dashboard; the
less-frequent OPDS feed link lives in Settings. The shelf selector stays
directly below the masthead. The `#`/A–Z rail remains sticky and horizontally
scrollable on wider screens, but scrolls naturally on phones to preserve
vertical reading space. Audio shelves use one compact row for view, playback,
shuffle, and podcast management instead of stacking separate toolbars.

Series with covers is the initial browse state. Its representative image comes
from the earliest sortable volume in the series, and selecting it opens that
series in the remembered book list-or-cover layout. Series List uses collapsed
native disclosure rows and loads books only when a row opens. All Books uses
the same remembered book layout and loads in pages of 120. A compact View menu
holds the two layout preferences so they do not become permanent competing
buttons. Library, browse, and layout choices persist in the browser. Library
opens the last valid shelf, or the first configured shelf on a new browser,
rather than implicitly querying All shelves. A compact native shelf selector,
search, and the active alphabet bucket apply consistently to the current state.
The selector scales to many libraries and uses the platform picker on phones
rather than growing a horizontal chip row.

Compact controls use 8–14px radii, while settings fields remain underline-only.
Depth is reserved for cover hover lift; lists use hairline rules. Every
interactive element has an apricot two-pixel focus outline with a
three-pixel offset. Theme follows the device on first visit, then persists an
explicit choice in local storage. Motion is limited to 200ms color and cover
lift transitions and is removed when reduced motion is requested. The reader
keeps its own edge-to-edge canvas and does not inherit the catalogue masthead.

## Android EPUB reader interaction boundary

The Android EPUB reader is an edge-to-edge reading surface with transient,
minimal chrome. Highlight is not a permanent toolbar action: it appears only in
Android's native text-selection action mode after Readium verifies that the
selection contains text. Creating a highlight clears the selection and returns
the reader to the book.

Saved highlights are managed alongside chapters and bookmarks in the existing
Contents bottom sheet. A note belongs to its highlight and is displayed and
edited inline within that highlight row; do not introduce a separate notes
screen, modal editor, or permanent reading control. Use compact, accessible
icon actions for adding or editing a note and deleting the highlight so the
reader's top row remains reserved for book-level navigation and tools.
Selecting the highlight text navigates to it. Removing a highlight requires a
native confirmation dialog that identifies the excerpt when available and
explains that removal applies to synchronized devices.

The inline note editor must remain visible when the software keyboard opens.
The Contents sheet accounts for IME and navigation-bar insets, and a focused
editor scrolls into view rather than being obscured or forcing the user out of
the highlight context. Save and Cancel remain explicit actions in the row.

Annotation conflicts remain in Contents under **Needs review**. Each conflict
shows the version on this device and the version from another device, including
available chapter and update-time context, before offering **Keep mine** or
**Use other**. When visible content matches but synchronization history differs,
say so explicitly rather than presenting two apparently identical choices
without explanation. An unresolved conflict disables note editing and highlight
deletion for that annotation at both the interface and state boundaries; the
reader must choose a version before changing or deleting it.

Synchronization and save status is exposed to TalkBack as a polite live region,
so updates are announced without interrupting reading. The announcement channel
must remain active when transient visual chrome is hidden; visible status copy
may stay inside the chrome, but accessibility feedback cannot depend on its
visibility. Readium highlight decorations remain theme-aware: warm apricot in
dark mode, deeper amber in sepia, and burnt apricot in light mode. Reapply
decorations when reader preferences change so highlights remain legible without
introducing a second annotation styling system.

## Installation and current first-run experience

Mochil targets three installation levels:

1. A portable release binary for Windows, macOS, and Linux.
2. A container image plus Compose file for NAS and home-server users.
3. Native NixOS and systemd modules for declarative deployments.

Container and systemd deployments currently copy a read-only template into
writable persistent storage on first startup. The authenticated Settings page
then edits library definitions and scan intervals using validated forms and
atomic writes. Reader credentials remain environment-managed.

The longer-term setup wizard roadmap is:

```text
start Mochil
      |
      v
open /setup with one-time setup code
      |
      +-- choose Local folder or Calibre library
      +-- add Books, Light Novels, Manga, Comics
      +-- choose metadata inference behavior
      +-- test access
      +-- create reader username/password
      |
      v
copy OPDS URL or scan QR code
```

When no configuration exists, native binaries bind setup to loopback by
default. Container deployments require a random one-time setup code printed to
the container log or supplied through `MOCHIL_SETUP_TOKEN`. Setup endpoints
are permanently disabled after initial configuration unless explicitly
re-enabled from the authenticated administration page.

The planned administration expansion will show library health, item/error counts, last
scan time, a Rescan action, and the exact OPDS URL. It will edit configuration
through validated forms and write it atomically with a versioned backup.

Release automation must produce checksummed archives, container images with
immutable version tags, an SBOM, and clear upgrade notes. Configuration and
cache formats require automatic forward migrations. An upgrade must never
modify canonical library files.

## Architecture

The staged stabilization and multi-media expansion plan is maintained in
[`docs/ARCHITECTURE_ROADMAP.md`](docs/ARCHITECTURE_ROADMAP.md). New media and AI
work follows that sequence so the book catalogue remains a tested, independent
module.

```text
                        OPDS catalogue engine
                  +--------------------------------+
                  | libraries, A-Z views, search, |
                  | metadata, ordering, pagination |
                  +---------------+----------------+
                                  |
                          storage interface
                    +-------------+-------------+
                    |                           |
             filesystem backend          Calibre backend
                    |                           |
             EPUB/CBZ files             metadata.db + files
```

Storage backends are deliberately unaware of OPDS XML. The catalogue engine
is deliberately unaware of local paths.

## Multiple libraries

Every item belongs to exactly one configured library. Library IDs are stable,
URL-safe identifiers and form part of item IDs and routes. Names are display
labels and may change.

```text
/opds
/opds/libraries/{library}
/opds/libraries/{library}/series
/opds/libraries/{library}/authors
/opds/libraries/{library}/titles
/opds/libraries/{library}/recent
/opds/libraries/{library}/search?q=...
/acquire/{library}/{item}
/stream/{library}/{item}?page={pageNumber}
```

One broken or unavailable library must not take other libraries offline after
startup. Each library has its own scan state and last error. The root feed can
therefore continue serving all healthy libraries.

Optional aggregate search/recent views may be added after per-library routes
are stable.

## Configuration

JSON is the initial configuration format so the binary remains dependency
free. Secrets are read from environment variables or files, never required in
the Nix store.

```json
{
  "listen": "0.0.0.0:8080",
  "public_url": "https://books.example.com",
  "scan_every": "10m",
  "libraries": [
    {
      "id": "books",
      "name": "Books",
      "kind": "books",
      "source": {
        "type": "filesystem",
        "path": "/srv/books"
      },
      "inference": { "folder_layout": "auto" }
    }
  ],
  "page_cache": {
    "directory": "/var/cache/mochil/pages",
    "maximum_bytes": 10737418240,
    "expire_after": "24h"
  }
}
```

Legacy `-library` operation creates one implicit library named `Library`.

## Normalized item model

```text
library_id       stable configured library ID
id               stable ID within the library
path             opaque backend-relative path
format           epub, cbz, pdf, cbr, ...
media_type       acquisition MIME type
title            undecorated publication title
authors          display author names
author_sort      sortable author names
series           optional series name
series_index     optional finite decimal number
description      bounded plain-text summary
language         optional language
identifier       ISBN/UUID/etc. when available
modified         source modification time
size             source byte size
page_count       page images, when known
cover            backend-specific cover locator
```

An absent series index is different from index zero. Series sorting is:

1. finite indexes in numeric order;
2. missing/invalid indexes;
3. normalized title as a tie-breaker.

Series feeds visibly prefix indexed items (for example `02 - Title`) because
OPDS 1.2 clients do not consistently understand extension metadata. The
undecorated publication title is retained in the normalized model.

## Metadata precedence

Higher-confidence data wins field by field:

1. explicit manifest values;
2. embedded format metadata;
3. configured folder/filename inference;
4. safe fallback values.

EPUB support includes EPUB 2/Calibre `calibre:series` and
`calibre:series_index`, plus EPUB 3 collection refinements
`collection-type=series` and `group-position`.

CBZ support reads `ComicInfo.xml`; otherwise image entries determine page
count and the first suitable page is the cover.

Inference profiles:

- `metadata-only`: never infer structural metadata.
- `series`: the containing folder is the series; numeric filename prefixes
  are series indexes.
- `auto`: infer a series folder only when multiple sibling files use numeric
  prefixes.

Recognized numeric prefixes include `01 -`, `1.5 -`, `v02 -`, `Volume 2 -`,
and `#12 -`. Filename author parsing is opt-in because `Title - Author` is
inherently ambiguous.

## Filesystem backend

The filesystem backend recursively discovers supported regular files without
following symlinks. It caches normalized metadata using path, size, and
modification time. Files are opened only after an acquisition ID is resolved
from the index, preventing arbitrary path access.

## OPDS and navigation

Each library provides Series, Authors, Titles, Recently Added, and Search.
Series, author-sort names, and titles are grouped into `#` and `A` through `Z`.
Leading English articles are ignored for buckets without altering display
values. Feeds are paginated with stable ordering.

Personal collections are lightweight book or dynamic-series references scoped
to one shelf. Library exposes them through a compact contextual control only
after a shelf is selected; OPDS groups them by shelf. Dynamic series include
newly scanned volumes automatically. Explicit series indexes remain
authoritative; missing indexes may be inferred from unambiguous volume markers,
and natural title ordering keeps `Volume 2` before `Volume 10` without renaming
source files. Nested side-story folders follow the main series run.

Every acquisition entry retains a standard OPDS download link even when page
streaming is also available.

## OPDS-PSE

Page streaming applies only to stable page-image formats. Initial support is
CBZ. Local CBZ files are indexed from the ZIP central directory and page image
names are naturally sorted.

The stream link advertises page count and a URL containing `{pageNumber}`.
The first implementation serves original JPEG, PNG, or GIF bytes and omits
`{maxWidth}`. Resizing is optional future work.

EPUB page streaming is out of scope because reflowable content has no stable
server-side page sequence. CBR and PDF streaming require additional extraction
or rendering dependencies and are later adapters.

## Audio shelves

Podcast, audiobook, and music shelves use Source Sans for dense 14px titles and
12px context rather than the catalogue's editorial display face. Rows keep one
primary play action; downloads, favourites, and queue actions live behind a
vertical three-dot menu. Podcast shows, audiobook series, and music albums form
quiet ruled groups. A tune icon discloses browsing and sorting, a centered play
control starts the current order, and an optional shuffle icon starts a random
queue. Artwork extraction is viewport-prioritized so FFmpeg workers serve
visible rows first. An intersection sentinel loads subsequent pages without
permanent pagination.

Podcasts browse by show rather than by episode. Cover mode presents square show
artwork; list mode uses native disclosure rows that are closed initially. Both
open or reveal only the selected show's episodes, keeping large subscriptions
scannable and avoiding a wall of individual episode controls.

The shared audio player is a 52px full-width bottom transport rather than a
floating card. Artwork is circular, carries the play/pause action, and gains a
thin progress ring; its glyph fades after interaction so the cover remains
legible. The remaining width belongs to title and context, followed by one
overflow. Selecting the title expands a focused bottom player with large real
artwork, precise seek rail, elapsed time, and previous/play/next controls.
Speed, shuffle, sleep, and close remain progressively disclosed.

Music opens as one ungrouped track list, initially bounded to 72 rows before
the existing cursor sentinel loads more. A compact browse selector switches to
Albums, Artists, or persistent account-scoped Playlists. Playlist creation is
inline and individual tracks join playlists from their existing overflow menu.

Podcast discovery searches Apple's public directory without requiring an API
key. Search artwork is proxied only from Apple's image host; subscribing stores
the resulting RSS URL in Mochil. Direct RSS entry remains available under a
secondary disclosure for private and unlisted feeds.

## Security and operations

- Treat library contents and manifests as untrusted input.
- Bound XML sizes, descriptions, decompressed page sizes, and cache usage.
- Reject archive traversal paths and symlinked filesystem items.
- Use constant-time Basic authentication checks.
- Use HTTPS whenever authentication or temporary links cross a network.
- Apply request, upstream, idle, and graceful-shutdown timeouts.
- Do not log authorization headers or generated download URLs.
- Provide `/healthz` for process health and eventually `/readyz` for library
  status.

## Delivery phases

1. Multi-library core and backward-compatible local EPUB support.
2. Backend interface, JSON configuration, independent scan state.
3. Complete EPUB series parsing and folder/filename inference.
4. (Retired) remote-storage acquisition redirects.
5. CBZ/ComicInfo, covers, and page counts.
6. Local CBZ OPDS-PSE.
7. (Retired) remote page cache.
8. Pagination, aggregate views, and optional additional formats.
9. Browser setup/admin UI and configuration migrations.
10. Signed release artifacts and NAS-specific installation templates.
