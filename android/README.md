# BoboBooks Android

This directory contains the native Android client. The Go server remains the
catalogue, authentication, streaming, and synchronization authority.

## Current foundation

- Native Jetpack Compose application using Material 3 and edge-to-edge insets.
- HTTPS server-address, username, and password sign-in.
- The password is exchanged once for a revocable device token and is never
  persisted by the app.
- Device tokens are encrypted with an Android Keystore AES-GCM key.
- Device-token authenticated library discovery and cursor-paginated book shelves.
- Server-backed Series and All Books modes with switchable cover/list layouts,
  dedicated series pages, and lazily loaded collapsible series rows.
- Per-mode layout and selected-library persistence, server-backed publication
  search, `#`/A–Z filtering, and native book detail screens.
- Compact, temporarily expanded search controls for publications and music.
- Native Home dashboard for Continue reading and Continue listening, with
  direct resume and compact Home/Library navigation.
- Authenticated exact-byte publication handoff to installed readers such as
  KOReader; book details remain a secondary overflow action.
- Minimal Library/Downloads/Settings navigation with the persistent player
  positioned above system navigation.
- Durable app-private publication downloads with explicit offline open/remove
  controls.
- Native CBZ reading with bounded archive access, paged manga navigation,
  right-to-left direction, continuous webtoon scrolling, and local page restore.
- Expandable full music player with artwork, seeking, previous/next, and
  background play/pause controls.
- Authenticated, disk-cached cover loading without putting tokens in image URLs.
- Adaptive cover grids that fetch the next page as the reader approaches it.
- Native music browsing with cursor pagination and authenticated artwork.
- Server-backed Artist → Album → Track and direct Albums/Tracks browsing with
  remembered cover/list layouts and complete group queues.
- Atomic app-private music downloads, local playback, and a downloaded-only
  view that remains available using cached shelf and track metadata offline.
- Media3 background playback with audio focus, headset-disconnect protection,
  notification/lock-screen controls, and a persistent in-app mini player.
- Native EPUB reader on Readium with search, bookmarks, highlights and notes,
  fonts, and reading-position sync with the server and KOReader.
- Read aloud in the EPUB reader (server narration module): a separate media
  service plays one server-made sentence at a time, so it keeps going with the
  screen off; the reader draws the sentence being read and offers voice, speed
  and a sleep timer. It is not offered on Android TV. A read-along EPUB (Storyteller) plays its own recording instead, streamed from the server and highlighted the same way; such a book downloads as text only.
- Adaptive BoboBooks launcher icon using the warm parchment/apricot identity.
- Source contracts separate catalogue, content, and reading progress. Supported
  sources are bounded first-party adapters for BoboBooks, local media, OPDS,
  podcast RSS, and read-only WebDAV; the app does not install or execute
  third-party extension code.

## Build requirements

- Android Studio with Android SDK 37
- JDK supported by Android Gradle Plugin 9.2

Open this directory as an Android Studio project, or build the debug APK with:

```sh
./gradlew :app:assembleDirectDebug
```

The generated APK is written to
`app/build/outputs/apk/direct/debug/app-direct-debug.apk`. The direct variant is
the self-hosted build and includes in-app update support. `assembleStoreDebug`
builds the Play Store variant without direct APK update installation. Neither
variant installs or executes third-party extension code.
The included Compose deployment mounts that artifact read-only and publishes it
through the authenticated Settings page. Other deployments can set
`OPDSHELF_ANDROID_APK` to the absolute path of an APK visible to BoboBooks.

The locally served `optimized` build reuses
`android/.signing/optimized.keystore` and refuses packaging if it is missing;
it never silently substitutes a machine-generated debug key. Back this
file up securely: replacing it changes Android's application identity and
forces users to uninstall before installing another self-hosted build. A build
host may instead set `BOBOBOOKS_OPTIMIZED_KEYSTORE_PATH` to the same persistent
debug-format keystore. This development identity is only for private,
self-hosted updates; public releases use the owner-controlled release key below.

## Signed releases

Production packages deliberately require an owner-controlled signing key. Keep
the keystore outside source control and provide all four values only to the
release environment:

```sh
export BOBOBOOKS_KEYSTORE_PATH=/secure/bobobooks-release.jks
export BOBOBOOKS_KEYSTORE_PASSWORD='…'
export BOBOBOOKS_KEY_ALIAS=bobobooks
export BOBOBOOKS_KEY_PASSWORD='…'
./gradlew :app:assembleDirectRelease
```

All release assemble/bundle tasks fail clearly when any value is missing or
the keystore path is not a regular file. The app never embeds a server address,
credential, or signing secret. Back up the keystore and passwords separately:
losing them prevents compatible updates to existing installations.

EPUB files now open in the native Readium reader with durable structural
locators and nearby text anchors, table-of-contents navigation, bounded search,
bookmarks, font sizing, Light/Sepia/Dark themes, and paged or scrolling
presentation. The next reader slice connects these locators to conflict-safe
KoSync interoperability, then adds highlights.

The complete mobile-first plan, including Series → Volumes browsing, the shared
collapsed-group interaction, Android TV, and Kobo compatibility boundaries, is
documented in [`../docs/ANDROID_ROADMAP.md`](../docs/ANDROID_ROADMAP.md).
