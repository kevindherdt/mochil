import test from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_FRAME, clampFrame, placeModel, sceneCss, scenes } from "../static/hermes-scenes.js";

test("every named scene draws something, the default and a missing picture draw nothing", () => {
  for (const scene of scenes.filter((s) => s.id)) assert.ok(sceneCss(scene.id).length > 20, scene.id);
  assert.equal(sceneCss(""), "");
  assert.equal(sceneCss("nonsense"), "");
  assert.equal(sceneCss("image", ""), "");
  assert.match(sceneCss("image", "/api/avatar/background?v=1"), /cover.*url\("\/api\/avatar\/background\?v=1"\)/);
});

test("a frame is kept inside what the server accepts", () => {
  assert.deepEqual(clampFrame({ zoom: 9, x: -5, y: "0.5" }), { zoom: 2.2, x: -1, y: 0.5 });
  assert.deepEqual(clampFrame({ zoom: NaN }), DEFAULT_FRAME);
  assert.deepEqual(clampFrame(undefined), DEFAULT_FRAME);
});

test("zoom scales the model about the middle and the offsets move it by up to half the frame", () => {
  const input = { width: 300, height: 380, modelWidth: 1000, modelHeight: 2000 };
  const base = placeModel({ ...input, frame: DEFAULT_FRAME });
  const zoomed = placeModel({ ...input, frame: { zoom: 2, x: 0, y: 0 } });
  assert.ok(Math.abs(zoomed.scale - base.scale * 2) < 1e-9);
  const right = placeModel({ ...input, frame: { zoom: 1, x: 1, y: 0 } });
  assert.ok(Math.abs(right.x - (base.x + 150)) < 1e-9);
  const down = placeModel({ ...input, frame: { zoom: 1, x: 0, y: -1 } });
  assert.ok(Math.abs(down.y - (base.y - 190)) < 1e-9);
});
