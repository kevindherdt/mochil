import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const staticRoot = new URL("../static/", import.meta.url);

// The "continue reading" card's hover glow is a colored box-shadow (a blurred halo).
// On a 16-grayscale e-ink panel a blurred box-shadow renders as a muddy smear instead
// of a crisp edge, so every other hover/elevation glow in this app (dashboard.css's
// .jump-card, music-library.css's .music-track-card) is scoped with
// :not([data-palette=eink]). .reading-card's glow in book-list.css must match that
// established convention.
test("reading card hover glow is suppressed for the e-ink palette like other card glows", async () => {
  const bookList = await readFile(new URL("book-list.css", staticRoot), "utf8");

  const hoverBlock = bookList.match(/@media \(hover:hover\) and \(pointer:fine\)\{\.reading-card:hover[^]*?\}\}/)?.[0];
  assert.ok(hoverBlock, ".reading-card hover rule under (hover:hover) and (pointer:fine) must exist");

  const glowRule = hoverBlock.match(/([^{}]*\.reading-card:hover)\{([^}]*box-shadow[^}]*)\}\}$/);
  assert.ok(glowRule, ".reading-card:hover must still set a box-shadow glow for pointer devices");

  assert.match(
    glowRule[1],
    /:not\(\[data-palette=eink\]\)/,
    ".reading-card:hover's box-shadow selector must exclude the e-ink palette, matching .jump-card and .music-track-card",
  );
});
