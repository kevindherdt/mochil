# Reader engine, sync, KOReader plugin and Android review (2026-10-26)

Scope: a read-through of `internal/web/koreader_plugin/main.lua` (545 lines), the plugin packager, the position bridge, and the Android reader/sync code and build config. Nothing here was run on a real KOReader device or an Android device. Items marked **verify** are suspicions from reading code only.

## 1. Is Foliate still the right web reader engine?

Keep it for now. Reasons:
- It is small, dependency-free, and handles EPUB, MOBI/KF8, FB2 and CBZ. We already patched it (range reading, a `.find` bug) and built Media Overlays read-aloud on it.
- Its own README still calls it incomplete: no continuous scrolling and an unfinished fixed-layout renderer. Our patches mean upgrading the vendored copy needs a manual merge.
- Alternative worth a spike: the Readium `ts-toolkit` navigator. It is actively developed, has audio/TTS/media-overlay support and matches the Readium toolkit the Android app already uses, so Android and web locators could share one model. Cost: a large migration of the reader, sync and annotation code. Licence, bundle size and fixed-layout quality were not checked.
- epub.js: no evidence of recent activity. Not recommended.
- Decision rule: only migrate if fixed-layout/comics or continuous scroll become real requirements. Otherwise spend the effort on the items below.

## 2. Sync status

What exists: per-device pairing tokens, revisioned reading positions with 409 conflicts, a text-anchor bridge (web, Android and KOReader positions convert through `internal/readpos`), KOSync-compatible endpoints, and highlight sync with tombstones.

Gaps:
- Never validated against a real KOReader device (Kobo `koboSpan` is also unhandled).
- Plugin progress pushes carry no base revision, so KOReader is last-writer-wins while Android and web use conflict detection.
- No automated end-to-end test across web, Android and KOReader. Only unit tests per piece.
- No claim is made here about parity with Bookorbit: its sync code was not re-read for this review.

## 3. KOReader plugin findings (`main.lua`)

Reliability and UX (highest value first)
1. **Blocking network on the UI thread at suspend and close** (`onSuspend`, `onCloseDocument`) with `LARGE_BLOCK_TIMEOUT`/`LARGE_TOTAL_TIMEOUT`. With Wi-Fi off this can freeze the device. Check `NetworkMgr:isConnected()` first and use a short timeout for background pushes.
2. `onReaderReady` pull plus highlight sync runs while the book is opening and can prompt Wi-Fi. Defer with `UIManager:nextTick`/`scheduleIn`, and add a setting to never auto-enable Wi-Fi.
3. `localAnnotations()` calls `annotationAnchor()` for every non-synced highlight on every sync, each doing about 48 `getNext/PrevVisibleWord` calls. Cache the anchor on the item (`bobobooks_anchor`) after first computation. Slow on e-ink.
4. `device_id` is `os.time()` plus unseeded `math.random`. Collisions are possible. Derive from the model plus a random 128-bit value read from `/dev/urandom`.
5. `request()` treats every non-200 as failure. Handle 409 on progress once the server supports base revisions for the plugin, and surface 429/5xx as retry-later.
6. Progress push is last-writer-wins and compares clocks (`updated_at` vs `bobobooks_seen_at`). Send `base_revision` and store the revision per document.
7. Highlight sync is rolling (EPUB/FB2) only. PDF/CBZ highlights are ignored on apply and have no usable locator. Document it or add page-based support.
8. Finished/abandoned state (`summary.status`) is not synced, so books finished on KOReader stay "reading" in Mochil.
9. Deletion detection needs two syncs (`missing >= 2`). It is safe but slow; consider using `onAnnotationsModified` diffs.

Packaging and maintenance
10. `_meta.lua` has no version, and `/status` does not report a minimum plugin version. Add `version` and a server-side "plugin is outdated" hint in Settings.
11. The token sits in plain `bobobooks_config.lua`. Acceptable for a device token, but Settings should list and revoke device tokens (verify it does) and the plugin should refuse non-HTTPS `server_url` unless explicitly allowed.
12. `buildKoReaderPlugin` iterates a map, so zip entry order is random. Sort entries and set `Method: zip.Deflate` and a fixed modified time for reproducible downloads.
13. Code is dense one-liners with mixed tabs and spaces and has no tests. Add `luacheck` plus a small Lua test harness with stubbed `ui`/`document` so anchor scoring (`apply`, `resolveAnnotationLocator`) is covered.

## 4. Android findings

Sync
1. `ReadingProgressClient` hand-rolls `HttpURLConnection` (10 s connect, 20 s read, no retry, no backoff) although OkHttp 5 is already a dependency. Move to one shared OkHttp client with retries and connection reuse.
2. Pending offline progress is only retried while the reader is open. Add WorkManager with a network constraint so progress syncs after the app closes.
3. The Android anchor is `version: 1` built from `locator.text.highlight`. That text is only reliably non-empty when something is selected. **Verify** on a device that a plain page position yields a non-empty `exact` (web and KOReader send version 2 with about 16 words). If it is empty, the bridge degrades to fraction-only and KOReader lands on the wrong page.
4. JSON is built with `org.json` even though kotlinx-serialization is a dependency. Move to serializable DTOs so the contract tests in `contracts/` can share types.

Build and security
5. **Update signing uses `androiddebugkey` with password `android`** (`optimizedPersistent`). Because self-update trusts this identity, use a real keystore with a secret password and verify the downloaded APK's signing certificate and hash before install.
6. `usesCleartextTraffic=false` blocks plain-HTTP LAN servers entirely. Many self-hosters use `http://192.168.x.x`. Add a per-host opt-in through a network security config, shown with a warning in Device Settings.
7. No `POST_NOTIFICATIONS` permission is declared. On Android 13+ the playback notification may not appear. **Verify** on a device.
8. `versionCode`/`versionName` are bumped by hand (247 / 0.16.86). Derive them from one source in the release script.
9. Readium is pinned at 3.3.0. Check for newer releases and the changelog for locator and highlight fixes.
10. `EpubReaderActivity` and `PdfReaderActivity` declare no `configChanges`. **Verify** that rotation and font-scale changes do not drop the reading position (the locator store should cover it, but test it).

## 5. Suggested order

1. Plugin: non-blocking suspend/close pushes, anchor caching, device id, base revision (items 1-6).
2. Android: verify the anchor (item 3), then WorkManager retries and the OkHttp client.
3. An end-to-end sync test harness (web, Android and a stubbed KOReader plugin against one server).
4. Update-signing fix and cleartext opt-in.
5. Plugin versioning and `luacheck`/tests; sync finished state.
6. Revisit Readium ts-toolkit only if fixed-layout or continuous scroll is needed.
