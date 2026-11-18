# Security audit

Audit date: 2026-09-01

Scope: browser sessions, OPDS Basic Auth, login throttling, CSRF posture, proxy
headers, persisted state, file delivery, filesystem confinement, podcast SSRF,
request limits, media processing, and reachable Go dependency vulnerabilities.

## Fixed findings

- **High — login throttling bypass.** The client bucket trusted arbitrary
  `X-Forwarded-For` and included the TCP source port in the fallback address.
  An attacker could therefore rotate a header or connection and avoid the
  limit. Buckets now use the normalized direct peer IP. Forwarded headers do
  not influence throttling.
- **High — OPDS Basic Auth had no attempt limit.** OPDS/file clients now use the
  same bounded failure tracker as browser sign-in and receive `429` with
  `Retry-After` after ten failures.
- **High — reachable image and text dependency vulnerabilities.** The scanner
  found malformed-input allocation, panic, and infinite-loop paths reachable
  through WebP cover processing and PDF metadata parsing. The build now uses Go
  1.25 with `golang.org/x/image` 0.45 and `golang.org/x/text` 0.41.
- **Medium — opaque scan failures.** Catalogue errors now retain one bounded
  diagnostic for administrator logs. This identified one malformed Calibre EPUB
  while keeping source files untouched.
- **Medium — unbounded Android JSON response reads.** Authentication, pairing,
  catalogue, collections, dashboard, audio, podcast, video, progress,
  annotation, and Wiki clients previously trusted the server to terminate a
  JSON body at a reasonable size. They now share a 16 MiB response ceiling and
  64 KiB error ceiling, while small authentication responses use a tighter
  1 MiB limit. JSON collections also fail closed at their actual API or store
  contracts: catalogue pages and nested authors, dashboard sections, audio
  groups/tracks/chapters, podcast pages, video pages/seasons/subtitles, music
  queues/playlists, collections, annotations, Wiki entries, and external-library
  records cannot inflate into unbounded object graphs after byte validation.
  Downloaded media and exports retain their separate
  streaming byte limits. Exact-ceiling and overflow regressions cover the
  shared reader.

## Verified controls

- Browser sessions store only random-token hashes in memory, expire after 12
  hours with a two-hour idle limit, use `HttpOnly` and `SameSite=Strict`, and
  set `Secure` when HTTPS is detected.
- Login forms use random, single-use, ten-minute tokens and bounded request
  bodies. Redirect targets are restricted to local paths.
- JSON mutation endpoints use bounded bodies and single-value decoding; strict
  handlers reject unknown fields.
- Persistent state loading requires regular files, applies per-store size
  limits, and rejects trailing JSON. Writes use atomic replacement, and audited
  stores roll memory back when persistence fails.
- Filesystem acquisitions re-resolve paths beneath the configured library and
  reject symbolic-link replacement.
- Podcast feeds, redirects, artwork, and media use a public-address-only dialer
  that rejects loopback, private, link-local, carrier-grade NAT, benchmark, and
  infrastructure address ranges, including after DNS resolution.
- Media and document requests use indexed identifiers rather than accepting raw
  filesystem paths. Large reads, archives, images, JSON, and external process
  output have explicit bounds.
- Login and application documents set CSP, frame denial, no-sniff, and a strict
  referrer policy. Application forms are restricted to the same origin, and a
  permissions policy disables unused camera, microphone, geolocation, payment,
  USB, and browsing-topics capabilities. Authenticated media uses private
  caching directives.

## Verification

- `govulncheck ./...` with `golang.org/x/vuln` 1.7.0: no reachable
  vulnerabilities; 18 advisories exist only
  in required modules along call paths Mochil does not use
- `go vet ./...`: pass
- Race-enabled account, catalogue, media, state, web, and wiki suites: pass
- Android audio and video revision caches include the opaque paired-device ID,
  preventing progress state from crossing an account change while the media
  service remains alive. Tokens are neither placed in cache keys nor logged.
- Android's encrypted Saved-OPDS cache now separates device-local publications
  from the current paired account mirror. Legacy unowned rows migrate to local
  storage only, and a reconnect cannot upload another account's pending titles.
  The stored account owner is an opaque digest of the paired-device ID.
- Authenticated OPDS artwork and WebDAV folder pages, media indexes, and stream
  references now carry a non-secret cache generation owned by the encrypted
  connection record. Address or credential edits rotate it atomically, so a
  process death before asynchronous cleanup cannot expose or reuse the prior
  source's derivatives. Cache filenames and stream tokens never derive from a
  password; simple display-name changes preserve useful cached work.
- Native Android Wiki offline snapshots, mirrored images, and active drafts are
  scoped by the opaque paired-device identity as well as the vault. Unowned
  cache formats from older builds are deleted on first access instead of being
  guessed into the current account; server Markdown and other canonical data
  are never touched by this cleanup.
- The encrypted audio-progress outbox records an opaque paired-device owner.
  Process restart replays only checkpoints created for the current credential;
  legacy unowned and previous-account rows are discarded before any network
  request, including explicit backward-seek retries.
- `npm audit --omit=dev --audit-level=high`: zero vulnerabilities
- Direct and Store Android manifest privacy contracts: pass
- Direct and Store Android unit/lint suites cover bounded response handling:
  pass
- Both optimized APKs and the Store App Bundle pass
  `check-store-readiness.sh`, including source, dependency, packaged hostname,
  executable-extension, and store-updater checks. Store APK permission
  inspection additionally rejects broad storage, package discovery, and
  package-installation permissions.
- The retired executable-source audit covers active Android source and flavors,
  Go/server source, browser assets, dependency declarations, included Gradle
  modules, Docker/version-control exclusions, both optimized APKs, and the
  Store App Bundle. Only the generic legacy-config field remains: decoding
  clears it before runtime assembly, Settings never returns it, and the next
  Settings save removes it from disk. Generated local Gradle reports are not
  source, image context, or package input. The Store gate now also restricts
  that legacy field to its decoder and redaction owners and verifies both
  discard operations remain present.
- Spoofed-forwarding and rotating-source-port regression test: pass
- Production Go 1.25 container build: pass

## Remaining operational recommendations

- Keep Mochil reachable through the configured HTTPS reverse proxy and do
  not expose its plain HTTP container port directly to untrusted networks.
- Configure HSTS at the TLS-terminating proxy, where HTTPS availability is
  known. Adding HSTS inside an HTTP-capable application can lock out valid local
  deployments.
- Treat the Calibre library as single-writer storage. Mochil may read it
  concurrently, but metadata writes should remain explicitly enabled and
  coordinated with Calibre.
- Review security headers and dependency vulnerabilities during each release.
