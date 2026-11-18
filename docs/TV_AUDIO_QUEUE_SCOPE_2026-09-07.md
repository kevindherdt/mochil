# TV audio queue scope

Status: worktree change after published 0.16.75, not packaged.

TV library playback and Now Playing queue edits treated audiobooks as eligible for the saved music queue. The server explicitly accepts only music library items at that endpoint, so audiobook actions caused failed writes. The Now Playing guard also inspected only the first track, allowing mixed queues through.

The TV routes now replace saved music only when the complete nonempty playback queue is music. Audiobook play-next/append remain local playback actions and are not sent to the music queue endpoint. Mixed queues remain playable/editable locally without replacing the saved music queue. An empty general playback queue does not implicitly clear saved music; explicit music clear operations elsewhere are unchanged.

Pure policy tests cover music, audiobooks, podcasts, unknown types, empty lists, and mixed ordering. Both complete Direct and Store unit suites passed. The UI layout and local playback calls were not changed. Physical TV remote playback and visible sync-error recovery remain separate verification/work items.

## Current-player identity targeting

TV queue play/move/remove callbacks now pass the selected track rather than its captured row index. The playback ViewModel resolves the complete library/item identity against the controller's current media items before acting. A missing identity produces index -1, which existing mutation bounds treat as no local operation. Same-ID tracks from different libraries remain distinct. Legacy index overloads remain available for other callers.

Unit tests cover index shifts, vanished targets, empty queues, and cross-library IDs. Both full Android suites passed. This does not claim a physical remote test or durable cross-device queue reconciliation; focus-after-removal still uses the existing positional fallback and deserves a separate review.
# Live removal focus target

Post-0.16.75 worktree: removal focus selection now uses the latest queue snapshot and media identity, not the index captured when the context menu was composed. A no-op removal keeps the surviving target; a successful removal selects the next row at its live position, falling back to the previous row at the end. Tests cover reordered queues, missing targets, the last remaining item, and empty results. Both Android unit-test suites passed. Physical remote focus restoration still needs device acceptance; these tests verify target selection, not Compose focus delivery. Impeccable hardening guided this behavior-only change; layout and controls are unchanged.
