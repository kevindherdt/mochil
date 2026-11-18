# Offline wiki image planning

Status: published in 0.16.82/code243. Final release evidence is in `ANDROID_RELEASE_0.16.82_PREPARATION.md`.

The offline mirror runs from `viewModelScope.launch`. Network and storage operations already switched dispatchers, but the subsequent asset-index construction and parsing of up to 128 cached notes ran on the caller's UI thread. Image planning now runs on `Dispatchers.Default`.

The planner uses the existing Markdown parser directly, avoiding an unused heading outline and an intermediate filtered image list. Coroutine cancellation is checked before planning, between notes, during parsing, and during image resolution. This prevents a cancelled vault operation from needlessly parsing the remaining note content. Index construction itself remains a synchronous bounded operation within the background dispatcher.

Tests retain referenced-image deduplication coverage and add cancellation inside a long note and a zero-capacity fast path. Both full Android flavor unit suites passed via `/tmp/bobobooks-wiki-offline-planning-tests.log`. This verifies behavior, not a measured device frame-time improvement. No layout, offline capacity, media content, or storage format changed.
