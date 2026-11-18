# Mochil

Mochil is a self-hosted web library, edge-to-edge reader, and OPDS 1.2 server
for existing ebook and comic collections. It provides Calibre-like browsing
without moving files or making a database the source of truth.

```text
Mochil
  Books
    Series -> #, A ... Z -> series -> numerically ordered books
    Authors -> #, A ... Z -> author -> books
    Titles -> #, A ... Z -> books
    Recently Added
    Search
  Light Novels
  Manga
  Comics
```

Libraries may be ordinary directories or Calibre libraries. A single
self-contained Go server binary serves the catalogue and embedded web assets
without a database; the cache can always be deleted and rebuilt.

The product design is in [DESIGN.md](DESIGN.md). The phased modular-growth plan
and anti-monolith requirements are in
[docs/ARCHITECTURE_ROADMAP.md](docs/ARCHITECTURE_ROADMAP.md).
For a native Debian container deployment, see
[docs/DEBIAN_INCUS.md](docs/DEBIAN_INCUS.md).

## The smallest install

Three folders and one file. Copy [`example/compose.yaml`](example/compose.yaml) next to your media,
adjust the left-hand paths, and run `docker compose up -d`. It builds from the checkout today and
pulls `ghcr.io/OWNER/REPO:latest` once releases are published, so the file never has to change.

**What happens on first run, and why you do not need to read any of this first:**

1. The server starts with no account and no library.
2. The browser gets one page: **Create your account**. No environment variables are required, and
   the library is not served to anyone until that account exists.
3. After signing in, an install with no library goes straight to **Settings**, where a folder can be
   added. No JSON editing, no restart.
4. Public exposure: set `MOCHIL_USERNAME`/`MOCHIL_PASSWORD`, or keep the port on your own
   network. Leaving an install open on the internet is possible only by asking for it with
   `MOCHIL_ALLOW_OPEN=1`.

`scripts/install-smoke.sh` proves this path: it boots the image against empty volumes with no
configuration and checks that the app serves a usable catalogue, then that it comes back after a
restart. CI runs it on every push, and the release workflow runs it against the **published**
image tag, because the published image is what people actually install.

### What to back up, and how to upgrade

- **`config/config.json`** — your libraries and settings. Small, changes rarely.
- **`data/catalogs/`** — `accounts.json` (sign-ins, including the administrator), reading state,
  paired devices and KOReader sync. Also small, also irreplaceable: losing it means creating the
  account again and re-pairing every device.
- **the rest of `data/`** — caches and derived indexes. Safe to delete: the catalogue rebuilds by
  scanning, and covers and metadata are refetched. Back it up only to save a rescan.

So the two things to back up are `config/config.json` and `data/catalogs/*.json`.
- **`books/`** (your media) is yours, not the app's; the app never writes there unless you point a
  library at it for renaming or imports.

Upgrade: `docker compose pull && docker compose up -d`. Settings are read from the mounted
`config/`, so an upgrade keeps them. Nothing migrates silently — if a release ever needs it, the
release notes say so.

### Publishing a release

The image and the compose file are already shaped for it, so publishing is a sequence rather than a
project. When there is a remote and a version worth tagging:

1. Push the repo, then tag `vX.Y.Z` — the release workflow verifies, builds and pushes
   `ghcr.io/OWNER/REPO:vX.Y.Z` and `:latest`.
2. Make that package **public** in the registry settings, or `docker compose up` for a stranger will
   fail at pull with a permission error that looks like a typo.
3. The workflow then boots the *published* tag against empty volumes and runs the same first-run
   journey CI runs, so a broken image fails the release rather than the neighbour who tries it.
4. `example/compose.yaml` already points `image:` at that tag and carries `build:` as a fallback, so
   it needs no edit at release time.

## Easiest installation

The included Compose installation starts with a safe, empty catalogue on
Docker Desktop, Podman Desktop, or a NAS with Docker Compose:

1. Download this project.
2. Optionally create a root `.env` with `MOCHIL_USERNAME` and
   `MOCHIL_PASSWORD`.
3. Put media below this project's `media/` folder, or set
   `MOCHIL_MEDIA_PATH=/your/media/root` in `.env`.
4. Run `docker compose up -d --build`.
5. Open `http://server:8099/`. A new installation opens Library Settings so
   you can add local folders or Calibre libraries without editing JSON.
6. Add
   `http://server:8099/opds` to an OPDS reader.

No Go compiler, database, or library restructuring is required. Web-managed
settings are saved beside the Compose file in `./config/config.json`, with the
preceding successful version retained as `./config/config.json.previous`.
Reading state and disposable cover/catalogue caches remain in the Compose-managed
`mochil-data` volume. Back up both `./config` and that volume when recreating
or moving the installation.

Hardware video acceleration and serving a locally built Android APK are
optional host capabilities, not installation requirements. Keep them in a
private `compose.override.yaml`; the repository ignores that file. The default
Compose service works without `/dev/dri` and without an APK artifact.

Back up the complete application state before moving hosts. Using `docker
compose cp` avoids depending on Compose's generated volume name:

```sh
docker compose cp mochil:/data ./bobobooks-data
```

The config location can be changed without editing Compose by setting
`MOCHIL_CONFIG_PATH=/your/config/directory` in `.env`. On startup the
container prepares this small application-owned directory, then drops to its
unprivileged runtime user before reading configuration or serving requests.
It also repairs ownership of an existing regular `config.json` after a host
editor or restore changes its owner, preserving its file permissions. Media
ownership is never changed by this startup repair.

The settings view can add local-folder and Calibre libraries without manually
editing JSON. Saving validates the complete candidate configuration, scans its
libraries while the current catalogue remains available, and atomically publishes
the new runtime without restarting the container. A changed listen address still
requires a process restart. Unchanged shelves retain their in-memory catalogue;
only added shelves or changed source/inference settings are rescanned.

Before publishing installation changes, exercise the same empty, credential-free
Compose path a new self-hoster receives. The test uses a temporary config bind
and never reads or changes your real `./config` directory:

```sh
./scripts/compose-install-smoke.sh
```

The smoke test uses an isolated Compose project and temporary data volume. It
verifies the health endpoint, the actual first-run Settings shell, and a zero
restart count. It then restarts with a persisted library whose storage has been
removed and verifies that Mochil still opens so the library can be repaired
or deleted in Settings. The test also verifies that the stale library remains
present in the Settings API and that recovery does not trigger a restart loop.
Test state is removed automatically.

Installation follows a recovery-first rule: optional media, hardware devices,
Android packages, and remote services may add features, but their absence must
not prevent the settings interface from starting. A missing or disconnected
library is reported as a shelf problem while the server and other libraries
remain available. Persistent application state lives under `/data`; media
mounts are replaceable inputs rather than boot dependencies.

## Implemented

- Multiple independently indexed libraries
- Cover-led responsive web library and search
- Embedded Readium EPUB/CBZ reader with a settings dialog (text size, tap zones, read aloud); the older classic reader is being retired
- Validated browser settings for libraries and scan interval
- Persistent resized cover cache with configurable LRU size limit
- Backward-compatible single-folder operation
- EPUB 2, EPUB 3, and Calibre-library metadata and covers
- EPUB 3 series collection refinements and numeric positions
- CBZ and `ComicInfo.xml` metadata
- Folder and numeric filename inference
- Article-aware `#` and A-Z navigation
- Nullable decimal series indexes and visibly ordered series feeds
- Authors, Series, Titles, Recent, and OpenSearch feeds
- Local EPUB/CBZ acquisition with HTTP range support
- Read-only local document shelves
- Local CBZ OPDS-PSE page streaming
- Optional HTTP Basic authentication
- Atomic disposable caches and stale-cache startup fallback
- Nix flake, hardened NixOS module, and scratch container image
- Home with "Jump back in", collections and smart collections (saved searches by author, series, text, or series with missing volumes)
- Music, audiobooks, podcasts and video with queue, playlists, favourites, chapters, resume and a full-screen player that takes its colour from the cover
- Appearance settings: five themes (Espresso, Kiln, Nocturne, Paper, e-ink), light/dark, seven optional page backgrounds, and a choice of sign-in background
- Sign-in with "Remember me" (30 days) and browser password filling
- KOReader sync (progress) plus a KOReader plugin that browses and downloads from the library
- Android app (pairing, background playback, TV), distributed from the server

KOReader downloads are associated with the server and catalogue entry. Existing
files without a server-specific ownership record (including downloads made by
older plugin versions) are left untouched and are not automatically recognized;
a new download uses a numbered filename instead. You can still open the old file
and its KOReader reading data from the file browser. Changing the server address
also causes a fresh download rather than guessing that the old file belongs to it.

### Optional modules

Everything below is switched on in Settings > Modules, is off by default unless
noted, and does nothing while off. A failing module does not stop the rest.

| Module | What it adds |
| --- | --- |
| Video | Movies, TV and anime streaming, subtitles and resume |
| Audio | Audiobooks, podcasts and music |
| Documents | Documents and the wiki vault |
| Narration | Read a book aloud with a local voice (needs the voice runtime) |
| Library editor | Spreadsheet-style editing, import, quality and ISBN checks, collections, move/copy between libraries |
| Hardcover shelf sync, reading progress, ratings | Send reading status, page progress and star ratings to your Hardcover account |
| Android app | Pairing, APK download and the API the app uses (on by default) |

## Quick start: one local library

Go 1.25 or newer:

```sh
go build -trimpath -ldflags="-s -w" -o mochil ./cmd/mochil
./mochil -library /srv/books -cache /var/lib/mochil/catalog.json
```

Add this URL to an OPDS reader:

```text
http://server:8080/opds
```

Legacy flags and their environment equivalents:

| Flag | Environment | Default |
| --- | --- | --- |
| `-listen` | `MOCHIL_LISTEN` | `:8080` |
| `-library` | `MOCHIL_LIBRARY` | `./books` |
| `-cache` | `MOCHIL_CACHE` | `./mochil-cache.json` |
| `-scan-every` | `MOCHIL_SCAN_EVERY` | `10m` |
| `-public-url` | `MOCHIL_PUBLIC_URL` | unset |
| `-username` | `MOCHIL_USERNAME` | unset |
| `-password` | `MOCHIL_PASSWORD` | unset |

Docker media settings are configured through `.env`. `MOCHIL_HLS_MAX_HEIGHT`
sets the transcoding height ceiling (`1080` by default, or `0` to preserve the
source resolution), while `MOCHIL_HLS_CACHE_MB` bounds the persistent HLS
segment cache (`4096` MiB by default). `MOCHIL_VIDEO_PREVIEW_CACHE_MB` bounds
the independently queued trick-play frame cache (`512` MiB by default). Artwork and podcast download limits are
also documented in [.env.example](.env.example).
`MOCHIL_AUDIO_HLS_CACHE_MB` separately bounds seekable AAC compatibility
segments for local audiobooks and music (`2048` MiB by default); original files
remain the default direct-play and download sources.
Music and audiobook view settings also offer explicit background loudness
analysis for files without ReplayGain metadata. It is off until requested and
never rewrites source files.

### Phone layout check

`npm run test:browser:stability` drives a phone-sized Chromium against a running
server with a music and a book library (same `BOBOBOOKS_BASE_URL`,
`MOCHIL_USERNAME` and `MOCHIL_PASSWORD` as below). It fails if tapping a
track moves the page, the music options sheet leaves the screen, the back arrow
in an opened series is covered, or the A-Z strip shows inside a series.

### Browser regression check

The browser smoke test renders Dashboard and Library in Chromium at desktop
and mobile sizes in both themes. It fails on runtime errors, page-level
horizontal overflow, undersized mobile controls, or loss of the active audio
player while navigating to Settings.

With Mochil already running locally:

```sh
npm install
MOCHIL_USERNAME=reader \
MOCHIL_PASSWORD='replace-this' \
npm run test:browser
```

Set `BOBOBOOKS_BASE_URL` for a different address and `CHROMIUM_PATH` when
Chromium is not installed at `/usr/bin/chromium`. Set
`BROWSER_SMOKE_ARTIFACTS=artifacts/browser-smoke` to retain screenshots. The
Playwright package is development-only and is not included in the Mochil
binary or container image.

### Authentication

The browser interface uses an expiring, server-side session with an explicit
sign-out action. Opening the site does not emit a browser-native Basic Auth
challenge. Sessions expire after 12 hours, or after two hours without activity,
and are stored only in memory; restarting Mochil signs browsers out.

OPDS catalogue, acquisition, cover, document, and page-streaming endpoints
retain HTTP Basic Auth for compatible reading clients. Use the same configured
username and password when adding the OPDS feed to a reader.

## Multiple libraries

Copy [mochil.example.json](mochil.example.json), remove sources that are
not needed, and start with:

```sh
MOCHIL_USERNAME=reader \
MOCHIL_PASSWORD='replace-this' \
./mochil -config /etc/mochil/config.json
```

Minimal local configuration:

```json
{
  "listen": "0.0.0.0:8080",
  "cache_directory": "/var/lib/mochil/catalogs",
  "libraries": [
    {
      "id": "books",
      "name": "Books",
      "kind": "books",
      "source": { "type": "filesystem", "path": "/srv/books" },
      "inference": { "folder_layout": "auto" }
    },
    {
      "id": "manga",
      "name": "Manga",
      "kind": "comics",
      "page_streaming": true,
      "source": { "type": "filesystem", "path": "/srv/manga" },
      "inference": { "folder_layout": "series" }
    }
  ]
}
```

Valid library kinds are `books`, `comics`, `manga`, `mixed`, `movies`, `tv`, `anime`, `audiobooks`, `podcasts`, and `music`. Local video shelves support direct playback, bounded compatibility HLS, seasons, subtitles, artwork, and resume. Local audio shelves read embedded tags and support range streaming, downloads, resume, playback speed, a sleep timer, and Continue Listening. Audiobooks expose embedded chapters in a seekable chapter drawer and use natural series/volume ordering. Music shelves provide remembered track, album, artist, favorite, and playlist views with persistent queues. Podcast shelves show every subscription from the compact feed index and load episodes only after a show is opened; feed refresh retains the last good episode list when a provider fails. Episode progress supplies durable played state, with All, Unplayed, and Played filters plus explicit mark-played actions. Per-show automatic downloads are off by default and can retain 1–50 newest episodes within the globally bounded podcast cache; its live usage is shown in Manage. Server-side podcast feed, artwork, stream, and download requests resolve and dial public addresses only, including after redirects, so subscriptions cannot reach loopback, private, link-local, or infrastructure metadata-service networks.
Manga uses the publication/comics pipeline and defaults the web reader to
right-to-left page progression while still allowing a per-browser override.
Folder layouts for publication libraries are:

## Calibre libraries

Choose **Calibre library** under Settings → Libraries and enter the mounted
Calibre library folder. Mochil reads each book's `metadata.opf` and
`cover.jpg`. When a book has several formats, Mochil selects one in
this order: EPUB, AZW3, MOBI, PDF, then CBZ.

In a container, mount the Calibre folder read-only and use its container path:

```yaml
volumes:
  - /srv/Calibre Library:/media/calibre:ro
```

Then enter `/media/calibre` as the Calibre library folder.

Calibre shelves are read-only by default. To apply reviewed metadata repairs,
install `calibredb`, mount that particular library read-write, and enable
**Allow confirmed Calibre repairs** for the shelf. Mochil uses Calibre's
supported CLI rather than editing `metadata.db`, `metadata.opf`, or managed
book files itself. It resolves the exact Calibre record from Calibre's reported
format path, stores the original OPF in `/data/metadata-backups`, verifies every
change by reading it back, and rolls the job back on failure.

For Compose, an optional Calibre-enabled image is provided (it is larger than
the normal minimal image):

```sh
docker compose -f compose.yaml -f compose.calibre.yaml up -d --build
```

Mount only the Calibre shelf that you explicitly want Mochil to repair
without `:ro`. A custom executable location can be supplied with
`MOCHIL_CALIBREDB`; the default is `calibredb` on `PATH`.

Folder layouts for ordinary publication libraries are:

- `metadata-only`: trust embedded metadata only.
- `series`: always treat containing folders as series; use a numeric filename
  prefix as the position when present.
- `auto`: require at least two numbered siblings before inferring a series.

`filename_author: true` additionally enables the ambiguous
`Title - Author.ext` convention.

## Metadata and ordering

Embedded metadata takes priority. Ordinary folders remain useful:

```text
Mistborn/
  01 - The Final Empire.epub
  02 - The Well of Ascension.epub
  03 - The Hero of Ages.epub
```

Indexes are decimals rather than strings, so this order is correct:

```text
00.5 - Prequel
01 - First Book
01.5 - Novella
02 - Second Book
10 - Tenth Book
```

Items without an index appear after numbered items. Leading `A`, `An`, and
`The` are ignored for buckets, so `The Expanse` appears under E without its
display name changing. EPUB `file-as` is used for author navigation when
available.

## OPDS-PSE

Set `page_streaming: true` on a local comics library. Eligible CBZ entries then
contain both a normal download link and an OPDS-PSE stream link. Pages are
served in natural filename order (`1`, `2`, `10`) and remain bounded by a
decompressed size limit.

Mixed page image types and WebP archives remain downloadable but are not
advertised for streaming because OPDS-PSE 1.2 requires one declared JPEG, PNG,
or GIF response type. EPUB is intentionally not page streamed.

## NixOS and Incus

Local development:

```sh
nix run . -- -library /srv/books -cache /tmp/mochil-cache.json
```

Import the module from the flake. For multiple libraries:

```nix
{
  services.mochil = {
    enable = true;
    configFile = "/etc/mochil/config.json";
    environmentFile = "/run/secrets/mochil";
  };
  networking.firewall.allowedTCPPorts = [ 8080 ];
}
```

For a local library, mount it read-only into the NixOS Incus guest:

```sh
incus launch images:nixos/unstable mochil
incus config device add mochil books disk \
  source=/srv/books path=/srv/books readonly=true
```

The OCI image supports legacy single-library mode by default:

```sh
podman build -t mochil -f Containerfile .
podman run --rm -p 8080:8080 \
  -v /srv/books:/books:ro \
  -v mochil-data:/data \
  mochil
```

## Operations and security

- `/healthz` reports process health.
- `/` serves the web library and reader; `/opds` remains the catalogue URL.
- Covers are normalized to JPEG, constrained to 600 × 900 pixels while
  preserving aspect ratio, and cached under `cache_directory/covers`. The
  default 512 MiB limit is editable in Settings; least-recently-used covers are
  evicted after new writes.
- Remote publications opened in the web reader stream to the browser and the
  persistent `page_cache` simultaneously. Reopens are served locally until the
  seven-day expiry or 4 GiB least-recently-used limit is reached.
- SIGTERM performs graceful HTTP shutdown.
- Both username and password must be supplied to enable Basic authentication.
- Put TLS in the reverse proxy before exposing credentials or downloads.
- Library files are never addressed by arbitrary request paths; acquisitions
  resolve an ID from the current index first.

### Editing a notes folder (Wiki)

Mochil runs as an unprivileged user: UID/GID 1000 with the provided Compose
file (65532 in the bare image). To let the Wiki create and edit pages in a notes
folder owned by someone else — for example an Obsidian vault synced to another
account — run it as that folder's owner so the pages it writes stay editable
elsewhere. Find the owner with `ls -ln`, then set `MOCHIL_UID` and
`MOCHIL_GID` (in `.env` or `environment:` in `compose.override.yaml`). On the next start Mochil hands its own `/config` and
`/data` folders to that user; your media is never touched.

## Licence

This project is free software under the **GNU Affero General Public License
v3.0** (AGPL-3.0). See [LICENSE](LICENSE) for the full text.

In plain terms: you may run, copy, change and even sell it, but if you offer a
changed version to other people over a network, you have to publish your
changes under the same licence. That is what keeps a hosted copy of this project
from becoming a closed product nobody can inspect or improve.

The licence covers the code. The name and the logo do not come with it — see
[TRADEMARK.md](TRADEMARK.md). AGPL-3.0 is the same licence GitHub lists as
"GNU Affero General Public License v3.0".
