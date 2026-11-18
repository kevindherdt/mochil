# Web playlist edits preserve sequential intent

Status: source-only after 0.16.79, next deployment batch.

Serializing HTTP writes was insufficient: remove/up/down actions computed their replacement lists before entering the queue. Two rapid removals could therefore write two snapshots of the same old list and resurrect the first removed item.

`editPlaylist` now evaluates a transform against the latest confirmed local detail inside the existing serialized mutation queue. An unloaded summary must successfully fetch detail before any replacement write; failed detail loads never send an empty list. Explicit full replacement (`savePlaylist`) retains its snapshot-at-call contract. HTTP write/publication logic is shared without introducing another queue.

Menu remove/up/down actions use the new intent operation. Drag reordering captures track and target identities at drop time, then locates them in the latest list when executing. A missing identity cannot substitute another track at the old index. Queue dragging is unchanged because its optimistic local state already updates synchronously.

This fixes same-client operation ordering, not cross-device concurrent playlist editing: whole-list PUTs still need server-side revision/operation semantics for that stronger guarantee. Existing server-atomic append remains unchanged.

Verification: new delayed-write regression confirms consecutive removals produce `[two, three]` then `[three]`, not a resurrected first track. Detail-load failure test confirms no write. Full frontend suite passed 540 tests before final drag wiring; final suite and desktop/mobile playlist browser fixture logs are `/tmp/bobobooks-playlist-intent-final.log` and `/tmp/bobobooks-playlist-intent-browser.log`. No visual layout changes or new APK/deployment.

The architecture gate initially rejected growth in `audio-library.js`. Queue/playlist drag-to-state bindings were extracted into `music-drag-bindings.js`, retaining low-level gesture handling in its existing module and state persistence in the state owner. The gate now passes without raising its limit. Full frontend suite passed 540 tests after extraction (`/tmp/bobobooks-playlist-modular-tests.log`); desktop/mobile playlist form fixtures passed at 1440px and 390px. Gesture behavior still merits a rendered drag regression beyond form coverage.

## Rendered drag regression

`npm run test:browser:playlist-drag` now passes in Chromium at 1440px using real HTML drag gestures, the production drag bindings, serialized playlist state and a delayed synthetic transport. Starting from `[one, two, three, four]`, dragging one after three and then two after four while the first PUT is pending writes `[two, three, one, four]` followed by `[three, one, four, two]`. Final DOM order matches, drag decorations clear, and no page errors occur. This closes the gesture-test gap above; it is an isolated feature fixture, not personal-library or complete desktop-shell acceptance. Touch uses existing menu actions, not this fine-pointer-only enhancement.
