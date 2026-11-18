# Android standalone sources

Mochil Android must remain useful without a Mochil server. Connecting a
server adds synchronized progress, annotations, metadata, and a self-hosted
catalogue; it is not a prerequisite for opening personal media.

## Architecture

The UI consumes normalized publications, audio, podcasts, and video from source
adapters. Browsing, search, downloads, readers, and players do not branch on a
specific backend.

```text
Android library UI / readers / Media3 players
                    |
          normalized media repositories
                    |
  +-----------------+------------------+----------------+
  |                 |                  |                |
Mochil API   local SAF folders    OPDS feeds      WebDAV
                                      |
                                 acquisitions

Podcast RSS is an audio source with its own subscription and update policy.
```

## Delivery stages

1. **Source-neutral core:** stable source IDs, capability flags, normalized
   records, and interfaces for browse, search, open/stream, download, and
   optional write-back. The existing Mochil clients become the first adapter.
2. **Local device mode:** open directly into an empty-but-usable local Home,
   without a login gate. Settings → Sources adds Mochil servers, local
   Storage Access Framework folders, and later OPDS/WebDAV connections.
   Incremental scanning discovers EPUB/PDF/CBZ/CBR, music, audiobooks, and video;
   the index is disposable and source files are never moved.
3. **OPDS:** OPDS 1.2 navigation, search, pagination, encrypted Basic
   authentication, and acquisition into the same download/reader pipeline are
   implemented, including authenticated artwork and OPDS 2 JSON catalogues.
4. **Podcasts:** direct RSS/Atom subscriptions, bounded scheduled refresh,
   streaming, downloads, played state, and cleanup without a server.
5. **WebDAV:** encrypted connections, bounded `PROPFIND` browsing, strict
   same-origin credentials, ETag-keyed local caching, and native publication,
   audio, and video opening are implemented read-only. Bounded whole-tree scans
   reuse folder validators, persist an encrypted disposable media index, and
   provide quiet indexed search. Audio and video probe byte-range support before
   playback, stream through an exact-origin authenticated Media3 source when
   supported, and retain a bounded private-download fallback otherwise.

Recently browsed WebDAV folders retain a bounded encrypted last-known-good
snapshot. Folder ETags are sent on refresh when the provider exposes them;
unchanged listings avoid XML transfer/parsing, while an unavailable provider
leaves its last usable folder visible with an explicit offline message.
6. **Source management:** add, disable, reorder, refresh, edit, and remove
   sources independently; suggest cross-source duplicates without silently
   merging progress or files.

## Safety and performance

- Credentials use encrypted Android storage and never enter URLs, logs,
  diagnostics, or reading-data exports.
- Scans are incremental and bounded; large media is never loaded
  wholly into memory.
- One unavailable source cannot block another source or app startup.
- Disconnecting or signing out of a server removes only that source and never
  locks the user out of local libraries.
- Local mode performs no server communication unless the user explicitly adds
  a source or sends diagnostics.
- Removing a source deletes only its disposable index and explicitly selected
  app downloads, never canonical files.

Android TV shares these adapters but keeps its ten-foot UI. A future Kobo client
can share OPDS/WebDAV and sync semantics while retaining native filesystem and
reader integration.

## Implemented foundation

The app now enters a local Home when no server credential exists. A Mochil
server can be connected as an optional source without changing the local-source
state. Local folders use persisted Storage Access Framework read access and a
bounded recursive discovery pass for supported publication, audio, and video
extensions. A bounded, disposable index restores saved shelves immediately and
refreshes them in the background. Unchanged documents are reused by size and
modification time; embedded title, artist, album, and duration metadata is
extracted only for changed audio and video. Discovered records share a compact local library, publications
open in the native reader pipeline, and local audio/video open in the Media3
players. Local audio queues survive service restarts. A connected server can be
removed independently while local folders and imported fonts remain intact.

OPDS 1.2 catalogues can now be added in standalone mode or connected-server
Settings. The adapter handles nested Atom navigation, direct and descriptor-based
OpenSearch, bounded pagination, same-origin credentials, and EPUB/PDF/CBZ
acquisitions through the offline download and native reader pipeline.
Authenticated catalogue artwork loads only for visible rows through a four-request
bound, sends credentials only to the exact catalogue origin, validates redirects,
caps each image at 12 MiB, and maintains a 256 MiB local cache. OPDS 2 JSON
catalogues now negotiate through the same connection and normalize publications,
groups, navigation, pagination, templated search, artwork, and acquisitions into
the existing source contract. Connected Home and library selectors now
expose one minimal Sources destination for OPDS and WebDAV instead of requiring
a trip through Settings. WebDAV libraries can now be added from
standalone or connected Settings, browse bounded remote folders, and download
EPUB/PDF/CBZ files into the existing native reader pipeline without modifying
the remote source. Audio and video use the same bounded local-copy path before
entering the Media3 background player or native video player, keeping WebDAV
credentials out of playback URLs.

External membership is now independent from downloaded filenames and copies.
OPDS results can be added to an encrypted device library; when a Mochil
server is connected, membership and revision-checked EPUB progress synchronize
through stable provider, source, remote-title, and remote-chapter identities.
The server contract records whether future source work executes on the server
or on a specific device, without treating device SAF folders as server paths.
The same Saved library is now visible from connected and standalone navigation,
and keeps OPDS acquisition details only in encrypted device storage so a
connected catalogue can reopen the title. Server Settings separately accept one
administrator-controlled source download root and quota; an empty root disables
server downloads, and a device-selected SAF folder is never interpreted as a
path on the server.
