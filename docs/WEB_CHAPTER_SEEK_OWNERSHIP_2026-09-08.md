# Chapter seek ownership and active deck

Status: deployed in the web-only batch documented in `WEB_CHAPTER_RELEASE_2026-09-08.md`. Android APK remains 0.16.82.

Chapter navigation previously assigned currentTime without calling the progress writer's explicit-seek claim. It also captured the initial audio element, despite the parent switching its active element during gapless playback. This could read/seek the inactive deck when later opening a chaptered item.

The controller now resolves the active element through a getter for chapter lookup and selection, and invokes the supplied progress claim before assigning the chapter position. The default fixed-element option remains for isolated callers. Both Previous controls share one callback that claims a restart seek when beyond five seconds; previous-item navigation remains unchanged.

Tests exercise an actual deck switch, chapter lookup, claim-before-seek ordering, unchanged inactive deck, playback on the current deck, ignored non-chapter clicks and both previous controls. All 560 frontend tests pass (`/tmp/bobobooks-chapter-seek-tests.log`), together with the module-size gate and desktop/mobile chapter browser checks. This is wiring/state coverage, not physical multi-device acceptance or an assertion that every sync case is solved.
