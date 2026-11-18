# Ingest and library management direction

This document records the intended Mochil behavior for scanning existing
libraries, matching metadata, and optionally ingesting new files. The default
must remain safe for self-hosted collections: scanning does not rename, move,
or rewrite source files.

## Product decisions

### Existing libraries stay observe-only by default

Mochil should tolerate imperfect existing layouts and combine folder names,
filenames, embedded metadata, sidecars, and provider identifiers. A user should
not need to reorganize a collection before it becomes readable.

File management belongs to a separate, explicit **managed ingest** mode. It
must use a dedicated writable ingest root, show the proposed match and final
path before committing, copy and verify before replacing anything, record a
reversible journal, and never reorganize an ordinary mounted library in place.
Read-only sources such as WebDAV and Seafile remain read-only.

### Recommended naming is a strong hint, not an admission requirement

The scanner should publish a concise naming guide and recognize these canonical
forms while retaining tolerant fallbacks:

```text
Books/Author/Series/Series - Vol 01 - Title [isbn-978...].epub
Books/Author/Standalone Title (2024) [isbn-978...]/Standalone Title - Author.epub
Manga/Series/Series - v001.cbz
Manga/Series/Series - SP01 - Side Story.cbz
Audiobooks/Author/Series/Series - Vol 01 - Title.m4b
Music/Album Artist/Album (2024)/01-03 - Track Title.flac
Movies/Movie Name (2024) [tmdbid-12345]/Movie Name (2024).mkv
TV/Show Name (2024) [tmdbid-12345]/Season 01/Show Name - S01E03 - Episode Title.mkv
```

Zero padding is recommended for stable natural sorting. Provider identifiers
and years are optional but sharply reduce ambiguous matches. Embedded tags are
still the primary source for music. TV seasons and episodes follow the widely
supported `Season 01` and `S01E03` convention. Specials use `SP01` for written
series and `Season 00`/`S00E01` for television.

### Deterministic evidence precedes remote matching

The identity pipeline should be shared, typed, explainable, and ordered:

1. Exact embedded or sidecar provider ID (ISBN, TMDB, TVDB, MusicBrainz, etc.).
2. Valid identifier extracted from filename/folder text.
3. Embedded file metadata and sidecar metadata.
4. Typed folder structure and explicit volume/season/episode tokens.
5. Conservative filename parsing.
6. Ranked provider candidates using normalized title, author/artist, year,
   series, index, language, duration, and format-specific evidence.

Every candidate receives an evidence breakdown and confidence score. A high
threshold may auto-accept; a middle score enters review; ambiguous or weak
matches remain unmatched. A provider result must never overwrite identity
merely because it was first. User corrections and locks outrank later scans.

AI is not required for the base pipeline. An optional premium matcher may help
break an already narrow tie or extract an identifier from a difficult image,
but it must be explicit, privacy-aware, and never silently override the
deterministic result.

### Metadata tools are library-specific

The web metadata workspace should open one library at a time and select an
editor appropriate to its media type:

- Books, light novels, manga, comics, and audiobooks: author, title, series,
  series index, edition/format, identifiers, side stories, cover, and duplicate
  format review.
- Music: embedded tags, album artist, album, disc/track, year, genre, artwork,
  quality, and missing/ inconsistent tag reports.
- Movies, television, and anime: identify/match flow based on title, year,
  season/episode, provider ID, artwork, and per-episode overrides, similar in
  responsibility to established media servers rather than a Calibre table.
- Podcasts: RSS/feed metadata and local playback state, not generic file-table
  editing.

Large libraries use server-side filtering, cursor pagination, selection by
stable identity, and bounded batch jobs. The client never loads every library
or every item into one editing table.

## Client and web UX requirements

### Seamless web browsing

The Android app uses lazy lists; the web equivalent is cursor/offset fetching
plus an intersection sentinel and a bounded DOM window. “Load more” remains an
accessible fallback only. All main books, series, music, podcast, audiobook,
video, and document views should prefetch before the sentinel reaches the
viewport, serialize requests, retain scroll position, release off-screen
artwork, and keep a bounded number of DOM nodes. Nested series expansions also
need this behavior instead of per-series manual buttons.

### Editable connections and address migration

OPDS and WebDAV connections must support rename, URL/path, username, and
credential changes. Editing preserves a stable connection ID so saved-library
references remain attached; changing endpoint identity invalidates remote page
and index caches but leaves downloaded files intact.

Connected Mochil clients must support changing the server address without
disconnecting first. The client validates the existing device token against the
new HTTPS origin before saving it. If the token is not valid on that origin, it
offers sign-in at the new address while retaining local downloads and reading
data.

This phase is implemented on Android. OPDS and WebDAV edits retain stable
connection IDs; WebDAV endpoint or credential changes cancel stale work and
invalidate disposable page, index, and stream-reference caches without removing
downloads. A connected device validates its existing token against a replacement
HTTPS address or signs in there while retaining device-local media and reading data.

### Progressive-disclosure settings

Web Sources, Reading Sync, and Metadata settings should share the same quiet
settings language as Android: compact category navigation, a short status
summary, collapsed advanced sections, one primary action per section, aligned
switches, and errors adjacent to the field that caused them. Removed executable
extension and Suwayomi experiments must not return to the store-facing surface;
Sources remain declarative OPDS, WebDAV, RSS, local media, and configured
Mochil storage adapters.

## Implementation order

1. Editable OPDS/WebDAV connections and safe server-address migration.
2. Close continuous-scroll coverage gaps, including nested series lists.
3. Redesign Sources and Reading Sync with progressive disclosure.
4. Introduce the typed evidence/match model and per-library metadata workspace.
5. Add preview-only ingest planning.
6. Add opt-in managed ingest after journal/recovery tests prove it safe.

The managed-ingest journal prerequisite is now implemented independently of an
executor. It uses bounded, versioned, atomic state; canonical relative paths;
monotonic copy/verify/commit/rollback transitions; and deterministic recovery
steps. Ambiguous or mismatched targets always stop for manual review rather than
being overwritten. The internal executor now proves the next boundary for
dedicated POSIX roots: a plan performs no media mutation, execution revalidates
source confinement and size, copies and SHA-256 verifies private temporary
files, and uses an atomic no-clobber link for commit. It never removes an
incoming file or replaces an existing managed target. Completed journal entries
rotate while active and recoverable work is retained. Safe crash recovery is
also executable: interrupted copies restart, fully digest-matched commits are
finalized, and any mismatched or partial multi-file commit remains untouched for
manual review. Administrator configuration is now present as a collapsed
advanced Storage section:
both non-overlapping roots and a bounded file limit are required together, and
invalid changes cannot replace the live application handler. Configuration
constructs the executor and cache-owned journal, while planning and execution
are now available through administrator-only HTTP operations. Planning creates
only a journal entry; execution and recovery require the exact transaction ID
and run as cancellable bounded jobs. The Metadata workspace presents those
operations as a compact two-stage workflow: review creates an immutable no-copy
plan, exact transaction-ID confirmation starts the copy, and an unstarted plan
can be cancelled without touching either root. Status polling reports the
durable terminal state while the bounded job continues independently of the
page.

The same collapsed workspace can inspect the incoming root without creating a
journal transaction. Discovery is cancellable, recognizes only an explicit
media-extension allowlist, skips symbolic links and oversized files, caps each
page at 100 items and each traversal at 25,000 entries, and supports stable
relative-path cursors plus publication, music, and video filters. Every result
shows the inferred identity and an editable deterministic target suggestion;
choosing one merely fills the review fields. The normal plan boundary still
revalidates the source and refuses an occupied destination.

The discovery list can collect up to 100 files into a preview-only batch. This
creates no journal entry and performs no copy: it checks every confined source
and unused destination together, including duplicate sources and
case-insensitive destination collisions that would be unsafe on portable
libraries. Conflicts stay adjacent to editable target paths. Any edit makes the
review stale, and only a newly clear preview exposes creation of one immutable
batch transaction. The existing exact-ID confirmation, digest verification,
atomic no-clobber commit, and recovery rules remain unchanged.

The naming recommendations align with the broad conventions documented by
Kavita for series folders, Jellyfin for video seasons/episodes and provider IDs,
and Plex for artist/album/track music organization, while remaining optional in
Mochil.
