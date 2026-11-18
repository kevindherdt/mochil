# Direct video capability

This record defines the first independently removable video slice. It is based
on the useful behavior observed in `otherapps/mochi`, without copying Mochi's
database, routers, player shell, transcoding stack, or compatibility layers.

## User-visible behavior

- A movie shelf indexes supported video files without entering publication,
  OPDS, metadata-review, or reader paths.
- Opening a compatible item uses the browser's native video element.
- Seeking uses HTTP byte ranges against the configured storage backend.
- Position and completion are account-scoped and survive browser/container
  restarts.
- Unsupported containers or codecs show a clear direct-play limitation. They
  are never silently handed to an unbounded transcoder.
- Merely selecting Library returns cheap shelf/family summaries. Video files,
  posters, and metadata load only after a movie shelf is selected.

## Inputs and outputs

Input is an immutable filesystem or remote-backend object with a stable library
ID, normalized relative path, size, modification revision, and media type.
Output is a versioned movie snapshot containing stable item identity, display
title, source revision, direct-play media type, and an acquisition reference.
Playback state stores item identity, position seconds, duration seconds,
completion, timestamp, and device identity separately from the snapshot.

## Ownership and dependencies

`internal/video` owns video extensions, item identity, movie title inference,
and later television grouping. It may depend on storage acquisition
capabilities, but never on `internal/web`, publication `catalog.Book`, EPUB
metadata, OPDS, KoSync, or Foliate.

HTTP handlers validate item IDs and ranges, call the video/acquisition service,
and translate its result. Browser player code remains a separate video module.
Disabling video registration removes its routes and navigation without changing
book behavior.

## Resource and failure boundaries

- Directory walks publish an immutable replacement only after completion.
- Remote scans are cancellable and retain the last known-good snapshot.
- Direct playback forwards at most one validated Range request and streams the
  response; it never buffers a complete video in application memory.
- Poster extraction is absent from the first slice. Later thumbnail work must
  use bounded workers, input bytes, decoded dimensions, execution time, and
  persistent cache size.
- Missing files, unsupported direct-play formats, unreachable storage, and
  malformed ranges fail per item and do not invalidate the shelf.
- Playback-state writes are versioned, bounded, and atomic.

## Initial direct-play scope

The server may stream MP4/M4V, WebM, Ogg video, and MOV with accurate MIME types
and byte ranges. Container support does not promise codec support: the browser
remains authoritative through `canPlayType`. MKV and other containers stay
indexed but report that transcoding or an external player is required.

## Deliberately deferred Mochi behavior

- FFmpeg, CPU transcoding, VA-API, HLS, and temporary segment storage;
- Jellyfin compatibility and device profiles;
- TMDB/TVmaze/AniList matching and automatic metadata mutation;
- embedded subtitle extraction, chapter probing, trick-play thumbnails;
- Chromecast/remote playback and service-worker media caching;
- television seasons/episodes, collections, and rich posters;
- deletion or reorganization of canonical source files.

## Acceptance tests for the first implementation

- stable identity across rescans and changed identity/revision after replacement;
- supported extension and MIME normalization, including uppercase extensions;
- deterministic natural title ordering;
- path escape and symlink rejection for filesystem acquisition;
- complete and partial byte-range responses without whole-file buffering;
- unsupported direct-play response without starting external processes;
- atomic snapshot replacement and last-known-good retention on scan failure;
- account isolation and atomic persistence for playback position;
- complete existing publication, OPDS, reader, KoSync, metadata, race, and vet
  suites remain green.

## Implemented boundary

The filesystem scanner, versioned snapshot, persistent last-known-good cache,
natural ordering, selected-shelf JSON API, secure request-time file validation,
HTTP range delivery, and native browser player are implemented. Playback state
is now account-scoped, atomically persisted, saved during playback and on
pause/close, and remains separate from book CFIs and KoSync state.

Seafile video shelves now implement the remote side of this boundary through a
read-only list-and-range adapter. The proxy validates a single browser range,
requires provider sizing and range metadata, bounds streamed bytes to the
indexed object size, and never buffers a complete movie. Remote scans retain
their last valid snapshot and accept only video extensions. Capabilities that
require a local seekable file—probing, HLS, subtitle extraction, generated
posters, and trick-play frames—remain explicitly local-only.

## Playback expansion requirements

- Entering browser fullscreen requests landscape orientation where the mobile
  platform permits orientation locking; leaving fullscreen releases the lock.
- Embedded audio and subtitle selection requires a bounded media-probe and
  delivery layer. The native HTML video element alone does not expose portable
  embedded-track switching. Follow Mochi's proven boundary: `ffprobe` produces
  cached stream descriptors, text subtitles are extracted as WebVTT, and HLS
  or remux/transcode sessions are created only when direct play cannot satisfy
  the selected tracks.
- Jellyfin compatibility is an adapter over the same video snapshot and
  playback store, not a second catalogue. It requires stable Jellyfin-safe IDs,
  authentication/device sessions, library and item DTOs, playback-info with
  media streams, direct stream/range endpoints, subtitle endpoints, and
  started/progress/stopped reporting. Unsupported compatibility routes must
  return explicit errors rather than partially impersonating a Jellyfin server.
- FFmpeg workers must be optional and bounded by concurrent jobs, execution
  time, output bytes, cache bytes, cancellation, and hardware policy. Direct
  play remains available when that capability is disabled or unhealthy.
