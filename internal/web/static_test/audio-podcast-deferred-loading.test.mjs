import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("opening Music does not eagerly download podcast management", async () => {
  const source = await readFile(new URL("../static/audio-library.js", import.meta.url), "utf8");
  assert.doesNotMatch(source, /^import .*podcast-manager/m);
  assert.match(source, /await import\('\.\/podcast-manager\.js\?v=5'\)/);
  assert.match(source, /kind==='podcasts'.*ensurePodcastManager\(\)/s);
});
