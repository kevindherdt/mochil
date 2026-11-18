# Performance opportunities

Measured on 2026-08-24 against the deployed local catalogue: 12,214 books,
1,417 music tracks, 409 audiobooks, 62 podcast episodes, 77 videos, and the
current reading state. Browser measurements use Chromium at 1440×900 against
the local production container. Network timings are warm-cache observations,
not internet latency simulations.

Android playback synchronization now caches the last acknowledged revision for
each item. During uninterrupted audio playback, a periodic save therefore uses
one revisioned PUT instead of a GET followed by a PUT; with the existing
15-second cadence this is approximately 50% fewer progress-sync requests in the
steady state. A GET remains on first contact and after a conflict, where audio
and video retry at most once and only when the local position is genuinely
newer. This is a network/request-count improvement, not a claim of a 50% total
application CPU reduction.

Estimates below are ranges rather than promises. “High” confidence means the
same operation was benchmarked or its transferred bytes were measured.
“Medium” means the affected work is measured but the proposed replacement is
not implemented. “Low” means profiling is still required before changing code.

## Gains completed in this pass

Android startup observability and maintenance now have truthful ownership.
Foreground session duration is no longer presented as startup time; mobile and
TV record a real activity-start-to-first-frame duration. Startup orphan-partial
cleanup runs on `Dispatchers.IO` instead of blocking `Activity.onCreate`.
Android TV also memoizes the visible focus-identity set across unrelated chrome
recompositions, avoiding repeated allocation and hashing across every loaded
card. The diagnostics allowlist now retains the already-instrumented music
discovery, recently played, authors, favorites, wiki, and Now Playing timings.
The next physical-device report can therefore identify a request or rendering
hotspot rather than producing an incomplete trace.

Android TV focus visuals now share one conditional modifier path across Home,
the library rail, alphabet rail, Settings categories, and compact controls.
Unfocused, unselected items no longer carry idle scale, z-index, or zero-width
border modifier nodes; only the focused control receives a graphics transform.
This extends the same bounded-card optimization already used by the virtualized
catalogue without changing remote focus behavior or card dimensions.
Only the remembered target card now launches a focus-restoration effect. Other
visible cards no longer start idle coroutine effects merely to discover they
are not the restore target.

Native Android Wiki pages no longer parse Markdown twice on the Compose main
thread. One cancellable `Dispatchers.Default` parse produces both the bounded
render block list and outline, removing duplicate work and avoiding a long
main-thread stall on unusually large vault pages.

Native Android Wiki reopening now has an encrypted recent-page fast path.
The cached tree and up to 128 recently viewed or explicitly mirrored Markdown
pages are bounded to 5,000 entries, two million characters per page, and a
three-megabyte total encoded payload; oldest pages are evicted until the payload
fits. Explicit mirroring uses four concurrent page requests rather than an
unbounded fan-out. Cached pages paint read-only
while the authoritative device-token request runs. Opt-in diagnostics distinguish
cache load/save duration from `load_wiki_tree` and `load_wiki_page` network
timings without recording any vault identity or content. Physical-device
before/after timings remain the next measurement gate.

Android JSON transport now bounds both encoded bytes and decoded item counts.
This closes the compact-response amplification case where a response below 16
MiB could still allocate an excessive number of Kotlin/JSON objects. Limits
match server contracts (for example 5 dashboard entries, 60 video/show rows,
80 podcast episodes, 1,000 playlists, and 10,000 queue or playlist entries),
so ordinary paging and large legitimate libraries keep their existing behavior.

Android catalogue reads now share Wiki's bounded canonical-redirect policy.
Books, Home, music, audiobooks, podcasts, and video may follow a canonical
same-origin Android API path or a standard HTTP-to-HTTPS upgrade without
turning populated shelves into failed empty views. The device bearer token is
retained only inside the versioned Android API on the same host and permitted
port; login pages, cross-host redirects, HTTPS downgrades, and chains beyond
three hops are never followed. Mutation requests remain non-redirecting and do
not preconnect as GETs.

Explicit Android Wiki mirroring now resolves only images referenced by the
bounded offline page set. It downloads at most 32 images, caps each at 3 MiB
and the set at 24 MiB, and stores them under opaque revision-derived names in
no-backup app-private storage. Cached URI discovery runs on the I/O dispatcher;
Wiki scrolling performs no filesystem probes. A partial refresh preserves an
existing same-revision image instead of deleting it.

Submitted Android diagnostics identify no current Android TV frame bottleneck:
on the Google TV Streamer, three Home first-frame samples average about 102 ms
(112 ms maximum) and three library samples average about 79 ms (148 ms
maximum), with no recorded failures. Mobile reports did reveal concurrent
library-summary requests during some foreground transitions. A process-wide,
five-second summary cache now coalesces an in-flight request by server and
device ID without retaining the device token; explicit refresh bypasses the
completed-value cache while still joining an already-running request. This
targets duplicate network work only and leaves catalogue contents and refresh
semantics unchanged.

The same reports showed Android Home opening three authenticated continuation
requests, with 95th-percentile samples of roughly 319 ms for reading, 371 ms
for listening, and 358 ms for watching. These requests were parallel, so their
latencies did not add directly, but each repeated proxy, authentication, and
connection work. The versioned Android API now exposes one bounded Home
snapshot containing at most five entries per lane. New clients use that single
request and retain the three-request path as an automatic compatibility
fallback for older servers. The next opt-in report records this as
`load_dashboard_home` so the physical-device improvement remains measurable.

OPDS and dashboard Continue Reading queries now select only their requested newest positions inside the account reading store. The OPDS root uses a zero-allocation existence check, and its acquisition feed is capped at 100 entries. A synthetic 100,000-position account selects the newest 100 in about 22.5 ms with roughly 69 KiB and two allocations on the deployment host.

| Operation | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| Rebuild 5,000 series from 50,000 books | ~106 ms, 751,105 allocations | ~74 ms, 325,271 allocations | ~30% faster, 57% fewer allocations |
| Filter and regroup 50,000 books | ~48 ms, 328,429 allocations | ~24 ms, 25,113 allocations | ~50% faster, 92% fewer allocations |
| Open podcast shelf feed metadata | 2,077,701 transferred bytes, ~9.8 ms locally | 5,104 bytes, ~4.4 ms locally | 99.75% fewer bytes, ~55% lower local request time |
| Podcast show entry | fetched every episode before rendering | one page of at most 500 episodes | bounded memory/network use |
| Build a sorted podcast episode snapshot | store read lock held through copy and O(n log n) sort | lock held only for the O(n) copy; sorting happens on private data | refreshes and policy writes no longer wait behind snapshot sorting |
| Open ordinary music shelf | eagerly transferred 123.3 KiB queue | queue omitted until Queue or Add to queue is used | 49% less initial music JSON in the measured account |
| Confirmed playlist mutation | write followed by a full playlist collection GET and decode | one serialized write plus a local validated update | removes one redundant request and full collection decode per action |
| Open audiobook shelf | 590.5 KiB for all 409 books | 2.7 KiB for 14 series summaries | 99.5% fewer initial JSON bytes |
| Visible audiobook card artwork (14 covers) | Individual sources reached 305 KiB | 319.3 KiB total; 11.2–43.2 KiB each | Large per-cover outliers removed; bounded 320×480 output |
| Open ordinary music track list | 120 tracks, 142.5 KiB JSON, ~4,040 DOM nodes | 72 tracks, 87.9 KiB JSON, ~3,031 DOM nodes | 38.3% less JSON, 25.0% fewer DOM nodes |
| Sticky-header scroll work | layout reads on every scroll event | layout-free, animation-frame-throttled scroll state | removes forced-layout risk during scrolling |
| Unchanged production image rebuild | ~28 seconds with repeated dependency downloads | ~0.7 seconds from cached layers | ~97% faster repeat build |
| Long All Books browse session | DOM grew with every loaded page | At most 600 rendered cards | Bounded layout, style, and memory cost |
| Long Music track browse session | DOM grew with every loaded page | At most 360 rendered rows | Bounded layout, style, and memory cost |

The podcast result comes from returning subscription summaries with an episode
count. Episode records already have a dedicated paginated endpoint, so removing
them from the shelf response does not remove functionality.

## Current page observations

| Page or request | Current measurement | Assessment |
| --- | ---: | --- |
| First 80 all-shelf series | 18.8 KiB, 5.5 ms | Already efficient |
| First 80 Light Novel series | 18.2 KiB, 3.1 ms | Already efficient |
| First 80 Calibre series | 19.3 KiB, 3.3 ms | Already efficient |
| Movie list | 3.7 KiB, 2.5 ms | Already efficient |
| TV/anime show summaries | 0.5–0.9 KiB, 2.3–2.7 ms | Already efficient |
| Music first page, 72 tracks | 87.9 KiB, 9 ms | Browser creates about 3,031 DOM nodes across the view; additional pages retain cursor loading |
| Music queue state | 123.3 KiB, ~8 ms | Now deferred until Queue or Add to queue is used |
| Podcast subscription summaries | 5.1 KiB, ~4.4 ms | Efficient after this pass |
| Audiobook overview, 14 series / 409 books | 2.7 KiB, ~4 ms | Series contents are loaded only when opened |
| Empty document shelf | 0.3 KiB, 2.4 ms | Already efficient |
| Visible audio artwork | Audiobook cards total 319.3 KiB for 14 covers (11.2–43.2 KiB each); warm requests average 20.6 ms in Chromium | Bounded and cached; cold generation averages 513 ms while the two-worker FFmpeg queue fills |

## Prioritized opportunities

### 1. Lazy-load music account state — implemented

The ordinary music track page previously fetched queue, favorites, and
playlists at entry. The queue alone was 123.3 KiB in the measured account. It
is now loaded on demand while favorites and playlist names remain available for
the ordinary track controls.

- Measured gain: 123.3 KiB and roughly 8 ms are deferred, removing 49% of the
  initial music JSON in this account.
- Correctness: Add to queue first loads the saved queue, so lazy loading cannot
  overwrite entries from another session.
- Confidence: high.

### 2. Add audiobook series summaries and paged focus — implemented

Audiobook entry previously loaded all 409 records because grouping happened in
the browser. It now loads 14 compact server-generated summaries and pages an
opened series through a versioned cursor.

- Measured gain: 590.5 KiB became 2.7 KiB, a 99.5% reduction in initial JSON.
- Opening the measured 15-book Dresden Files series transferred 67.7 KiB in
  2.8 ms locally; returning to the overview required no full-catalogue reload.
- Natural series/volume order, standalone books, search, cursor expiry, mobile
  layout, playback, and downloads retain their existing behavior.
- Confidence: high.

### 3. Generate smaller audio artwork variants — implemented

Audio cards now receive a 320×480 maximum JPEG and the expanded player and
media-session metadata receive a separate 720×1080 maximum JPEG. Both embedded
images and sidecar files use the existing bounded worker pool and disposable
LRU cache. Sidecar size and nanosecond modification time are included in the
cache key, so replacing an image does not leave a stale resized copy.

- Measured result: the 14 visible audiobook cards transfer 319.3 KiB total and
  range from 11.2 to 43.2 KiB. Previously observed 100–305 KiB individual
  covers no longer reach the card view.
- CPU tradeoff: cold generation averaged 513 ms per request in Chromium while
  work queued through two FFmpeg workers; warm-cache requests averaged 20.6 ms.
- Correctness: malformed work retains FFmpeg's 30-second timeout, the input
  sidecar cap remains 32 MiB, cache size remains 512 MiB by default, and
  matching requests are coalesced.
- Confidence: high.

### 4. Reduce initial music DOM size — implemented

The ordinary music track view now requests 72 items initially rather than 120.
The existing versioned cursor, intersection sentinel, and visible Load more
control fetch additional pages without changing row markup or interaction.

- Measured gain: initial JSON fell from 142.5 KiB to 87.9 KiB (38.3%) and
  desktop DOM size fell from about 4,040 to 3,031 nodes (25.0%). Local API time
  remained effectively unchanged at 9 ms.
- Mobile verification: the initial list contains 72 rows, the next cursor
  produces 144, playback opens normally, and the page has no horizontal
  overflow at 390×844.
- Correctness: audiobook pages retain their 120-item bound. Playlist,
  favourites, and queue views retain their account-state behavior.
- Confidence: high.

### 5. Stream or compact the persistent catalogue cache — profiled and deferred

Five benchmark runs confirm that decoding a synthetic 100,000-item, 6.3 MiB
JSON catalogue takes 108–110 ms, allocates about 43.0 MiB, and performs roughly
200,060 allocations. This affects startup/rescan, not ordinary navigation.

- Production evidence: the two large book caches in this deployment are 3.1
  and 4.7 MiB for 5,038 and 5,158 books. Across all 10 libraries, catalogue
  readiness spans about 1.4 seconds while stores initialize concurrently.
- Decision: retain bounded JSON. A compact binary schema would need migration,
  corruption recovery, forward compatibility, and duplicate implementations
  across book, audio, video, and document caches for no measured navigation
  gain and little startup value at the current scale.
- Revisit gate: profile again if a real cache approaches 50 MiB, catalogue
  readiness exceeds five seconds on the target server, or a shelf grows toward
  100,000 items.
- Confidence: high for the current deployment; deliberately no code change.

### 6. Recycle long book and music lists — implemented

All Books and the flat Music track view now recycle rows outside a measured
viewport window. Top and bottom spacers retain the full scroll range, while
previous-page loading restores content when someone reverses direction or
jumps into an alphabet bucket.

- Books retain at most 600 cards and remove 120 at a time.
- Music retains at most 360 rows and removes 72 at a time.
- Range status remains logical rather than reporting only the retained DOM.
- Albums, artists, series, and other grouped indexes remain unchanged because
  they do not currently create the same unbounded flat-list hotspot.
- Confidence: high; forward, backward, reset, and concurrent paging behavior
  have regression coverage and were exercised against the production catalogue.

## Areas that should not be optimized now

- Book and series page requests are already 3–6 ms locally with small payloads.
- Video summary endpoints are below 3 ms and under 4 KiB before artwork.
- Keyset pagination near the end of a 50,000-book index remains microsecond
  scale and allocation-light.
- Publication-cache hits are microsecond scale.
- State-store durability should not be weakened to remove filesystem syncs;
  progress writes are small, user-visible, and correctness-sensitive.

## Next measurement gates

Before implementing opportunities 1–4, record mobile performance with network
throttling and browser CPU throttling, then compare transferred JSON, largest
contentful paint, interaction latency, and DOM node count. Each change should
retain the existing Go benchmarks and add a payload-size or browser regression
test so performance does not drift back unnoticed.
