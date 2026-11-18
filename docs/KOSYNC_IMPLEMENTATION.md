# KoSync implementation plan

## Compatibility target

Mochil should implement the KOReader v1 progress-sync API at the server root:

- `POST /users/create`
- `GET /users/auth`
- `PUT /syncs/progress`
- `GET /syncs/progress/{document}`
- `GET /healthcheck`

Requests use `Accept: application/vnd.koreader.v1+json`. Authenticated requests
carry `x-auth-user` and `x-auth-key`; KOReader computes the key as the MD5 of
the account password before sending it. Registration expects `201` or `402`,
authorization expects `200` or `401`, and progress writes accept `200`, `202`,
or `401`.

The wire record contains `document`, `progress`, `percentage`, `device`,
`device_id`, an update timestamp, and optional metadata. Writes must be atomic
and scoped by account plus document. The timestamp, rather than the furthest
percentage, determines which state is newest; users may intentionally read
backward.

## Document identity

Binary matching must reproduce KOReader's `util.partialMD5` exactly. It hashes
up to twelve 1 KiB samples beginning at byte offsets `1024 << (2*i)` for
`i = -1..10`: 256, 1024, 4096, 16384, and so on. A short final sample is
included; sampling stops when an offset yields no bytes. Filename matching is
the MD5 of the basename and should remain an explicit fallback because
unrelated files can share a filename.

BoboSync keeps up to eight observed binary digests as aliases on one
publication record. A repackaged EPUB can therefore acquire a new KoSync
digest without invalidating devices that still hold the old bytes. New aliases
are linked only from an exact embedded identifier, exact title plus author, or
exact series/index plus author; filename inference remains a uniqueness-checked
fallback. If duplicate shelves still produce multiple candidates, an already
active edition may disambiguate them. Otherwise the server refuses to guess.

The Android client hands an authenticated, exact-byte cached publication to an
installed reader through a read-only content URI. KOReader therefore computes
the same partial-MD5 as Mochil instead of receiving a converted or rewritten
copy. Android never substitutes a catalogue ID for this document digest.

For Seafile books, compute and persist the binary identifier from the local
publication cache after the first complete reader/download transfer. Never use
the catalogue book ID as a KoSync document ID.

## Position interoperability

KOReader sends a page number for fixed-layout documents and a CREngine XPointer
for reflowable books. Readium exposes an EPUB CFI plus a normalized progression.
These location strings are not interchangeable. Mochil must not present a
percentage-only jump as exact synchronization.

The Android interoperability path is therefore:

1. Cache the original publication bytes and compute KOReader's exact partial-MD5.
2. Pull before rendering and compare server timestamp/revision with the last
   state accepted by this device.
3. Convert KOReader's XPointer to a Readium CFI before applying reflowable EPUB
   progress. Use page numbers for fixed-layout comics and PDF.
4. Store the native locator for every client alongside normalized progression;
   never replace an XPointer with a CFI on the wire.
   Persist a bounded text anchor (`before`, `highlight`, and `after`) plus the
   EPUB spine resource with every structural locator. Android obtains this from
   Readium's first-visible-element locator; the web reader and KOReader plugin
   use the same anchor shape.
5. Convert CFI back to XPointer for Android-originated updates, and enable the
   push only for EPUB structures covered by shared KOReader fixtures.
6. Serialize and debounce writes, flush on pause/close, and use a conditional
   revision so an older Android session cannot overwrite a newer KOReader state.
   Fixed-layout acknowledgements require the exact PDF/comic page; normalized
   percentage tolerance is reserved for reflowable renderer drift and must
   never make adjacent pages equivalent.
7. If both devices advanced from the same known revision, retain both states and
   ask which location to resume instead of silently choosing the furthest value.

The converter will be ported as an isolated, fixture-driven module rather than
embedded in reader UI code. Readest demonstrates that CFI/XPointer conversion
is feasible, but Mochil will validate the port against KOReader-generated
fixtures and protect writes against stale-device conflicts before enabling it.
Percentage remains a fallback hint only and is not a valid KOReader reflowable
`progress` value.

Resume resolution is shared by present and future clients: exact native
locator, verified nearby-text match, matching spine/chapter, then approximate
percentage. This contract is platform-neutral so a later iOS/iPadOS client can
use Readium Swift and the same server records; Apple TV remains a separate
navigation and media UX track.

Android now persists bounded per-client Readium locators on the shared reading
record and uses conditional revisions for every write. It pulls before opening,
prefers a newer Readium locator, and otherwise searches the EPUB for the saved
KOReader/web text anchor. A stale write pauses synchronization instead of
overwriting the other device. PDF and comic writes additionally compare the
authoritative page returned by the server before clearing local pending state,
which also protects Android when connected to an older server. Direct
CFI/XPointer conversion and an explicit
conflict-choice surface remain required before declaring full KoSync parity.

## Delivery phases

1. Add a persistent, migration-safe sync store with explicit file locking,
   atomic writes, bounded request bodies, and tests for account isolation.
2. Implement the five v1 endpoints and exempt only those routes from the
   catalogue's HTTP Basic middleware; KoSync headers remain mandatory there.
3. Add the exact partial-MD5 implementation with fixtures generated by
   KOReader and expose the digest to the authenticated web reader only.
4. Add pull-only web synchronization by percentage, with a visible prompt when
   remote state is newer and a device ID persisted per browser.
5. Build cross-reader location fixtures for representative EPUBs and implement
   bidirectional conversion. Enable browser pushes only after KOReader accepts
   every generated reflowable location.
6. Add conflict, offline queue, retry, registration-disable, credential
   rotation, rate-limit, and reverse-proxy tests before calling sync production
   ready.

## Highlights and annotations

Annotation synchronization is a separate revisioned resource, not part of the
page-position payload. KOReader emits `AnnotationsModified` and exposes the
current book's annotation collection, but uploading that full collection on
each page turn would waste CPU, memory, bandwidth, and flash writes.

The implementation contract is:

1. Use the same resolved publication identity and account boundary as progress.
2. Convert KOReader, Readium, and web-reader records into a bounded canonical
   annotation containing a stable ID, kind, quote, note, color, created/updated
   timestamps, native locator, and surrounding-text anchor.
3. Debounce modification events, hash the canonical snapshot, skip unchanged
   snapshots, and flush on close/suspend. Enforce item, text, and request-size
   limits on both client and server.
4. Store revisions and tombstones so deletions propagate and offline clients
   cannot resurrect removed highlights. Merge independent item changes; surface
   a conflict only when the same note changed on two devices.
5. Pull before applying local edits. Preserve each client's native locator and
   restore by native locator, then verified text anchor, then spine/progression.
6. Expose an account-scoped export and deletion API before enabling automatic
   annotation sync by default. The server now provides deterministic JSON
   export containing only the authenticated account and permanent deletion of
   that account's server copy behind the exact confirmation phrase
   `DELETE ALL ANNOTATIONS`. Client-facing controls still need to be added.

An explicit publication-scoped adapter can also render the authenticated
account's live annotations into a new Mochil Notes page. It omits tombstones,
preserves quotes, notes, chapters, and timestamps as bounded Markdown, refuses
empty or oversized exports, and never gives either persistent store a reference
to the other. Android and web controls for invoking the adapter remain pending.

The current plugin caches the publication digest, debounces automatic position
pushes, avoids resending an unchanged successful position, bounds metadata and
anchor payloads, guards JSON/network failures, and flushes on close or suspend.
The revision and tombstone server store is now present, with bounded
incremental pull/apply APIs available to authenticated Mochil clients and
paired KOReader devices. Android's merge and native-locator adapters are active;
KOReader annotation transfer remains disabled until its bidirectional adapter
is complete. An upload-only adapter would look like sync while silently losing
edits from other devices.

## Security boundary

Sync credentials are separate from Mochil HTTP Basic credentials. Store
only a slow hash of the already-MD5-derived KoSync key with a per-installation
salt; compare in constant time. Never return credentials through the settings
API or log auth headers. Registration should default off after the first
account is created, or be controlled explicitly in server settings.
