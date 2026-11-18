# Android phone + TV review (2026-10-26)

Source read: TV shell, library route/screen/grid, Home, pairing, readers (EPUB, comic, PDF) and the EPUB sync path. Not run on a device. Unit tests (862) and `lintDirectDebug` pass.

## Fixed in source (not yet in a built APK)

1. **Server fills in a missing text quote** (deployed). Samsung positions arrived with an empty quote, so web/KOReader opened a few pages off.
2. **App sends a usable quote** when nothing is selected (uses the text after the position).
3. **Last pages now reach the server when the reader closes.** The 1.5 s debounced save was cancelled on close, leaving the server behind until the book was reopened. `EpubLocatorObserver.flush` runs from `onPause`.
4. **EPUB reader had no remote control for its menu.** Contents, font, search and close only opened by touch. Select/Up/Down/Menu now open it and move focus in; Left/Right keep turning pages while the menu is hidden; Back closes the menu before leaving the book. Comic and PDF readers got the same keys, a Back that closes their controls, and a hide timer that restarts on key presses.
5. **TV library focus.** Opening a library or series left focus on the Back button until the user pressed Right. Focus now moves to the first card once content loads, unless the user has already moved elsewhere (search, mode menu) or focus was restored.
6. **Module switches apply to the Android routes** (`/api/android/v1/*`).

## Still open

- No retry of failed progress saves after the app is closed (WorkManager is not in the offline Gradle cache; needs a network build).
- `HttpURLConnection` in `ReadingProgressClient` instead of the OkHttp client already shipped.
- Update signing still uses the development key; changing it forces a reinstall, so it needs a decision.
- Plain-HTTP servers are blocked (cleartext off).
- TV Home shows watching/listening only, no "continue reading".
- Everything TV-related is verified by reading and unit tests only. Check on a device: library open lands on first card; Back peels series/menu levels; book menu via remote; L1/R1 library switching.
