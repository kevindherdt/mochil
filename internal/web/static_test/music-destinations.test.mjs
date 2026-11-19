import assert from "node:assert/strict";
import test from "node:test";

import { destinationAfterSwipe, railOverflow, railScrollTarget } from "../static/music-destinations.js";

test("music destination swipe follows the visible destination order", () => {
  assert.equal(destinationAfterSwipe("discover", -80), "albums");
  assert.equal(destinationAfterSwipe("artists", 80), "albums");
  assert.equal(destinationAfterSwipe("tracks", -80), "queue");
});

test("music destination swipe ignores short movement and stops at edges", () => {
  assert.equal(destinationAfterSwipe("albums", 40), "albums");
  assert.equal(destinationAfterSwipe("discover", 80), "discover");
  assert.equal(destinationAfterSwipe("queue", -80), "queue");
  assert.equal(destinationAfterSwipe("missing", -80), "missing");
});

test("music destination rail scrolls only its horizontal viewport",()=>{
  const rail={scrollLeft:100,clientWidth:300}
  // The item is scrolled in with room to spare, so the edge fade never covers the active tab.
  assert.equal(railScrollTarget(rail,{offsetLeft:40,offsetWidth:70}),8)
  assert.equal(railScrollTarget(rail,{offsetLeft:420,offsetWidth:80}),232)
  assert.equal(railScrollTarget(rail,{offsetLeft:180,offsetWidth:80}),100)
})

test("music destination rail exposes only real overflow edges", () => {
  assert.deepEqual(railOverflow(0, 300, 640), { before: false, after: true });
  assert.deepEqual(railOverflow(170, 300, 640), { before: true, after: true });
  assert.deepEqual(railOverflow(340, 300, 640), { before: true, after: false });
  assert.deepEqual(railOverflow(0, 640, 640), { before: false, after: false });
});
