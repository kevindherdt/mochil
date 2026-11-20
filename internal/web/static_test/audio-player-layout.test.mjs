import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const stylesheet = await readFile(
  new URL("../static/audio-player.css", import.meta.url),
  "utf8",
);

test("expanded player gives title, context, and source facts intrinsic height", () => {
  assert.match(
    stylesheet,
    /\.audio-player\.expanded \.audio-player-copy\{[^}]*height:auto;[^}]*grid-template-rows:repeat\(3,min-content\)/,
  );
});
