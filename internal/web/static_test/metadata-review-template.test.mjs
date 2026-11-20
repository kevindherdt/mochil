import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  metadataReviewHTML,
  mountMetadataReview,
} from "../static/metadata-review-template.js";

// The Metadata page is retired. It was only ever reachable as a section nested inside the Libraries page,
// which is why its header appeared above a row of library settings, and the settings it held were replaced
// by Import (naming and checking new files) and the library editor (editing existing metadata).
test("the retired metadata page renders nothing", () => {
  assert.equal(metadataReviewHTML, "");
  const document = {
    querySelector: () => null,
    insertAdjacentHTML() {
      throw new Error("mounted a retired page");
    },
  };
  assert.doesNotThrow(() => mountMetadataReview(document));
});

test("the two settings that survived it live on the Libraries page", () => {
  for (const [label, file] of [
    ["the metadata providers", "metadata-provider-settings-template.js"],
    ["the Hardcover shelf", "hardcover-shelf-settings.js"],
  ]) {
    const source = readFileSync(new URL(`../static/${file}`, import.meta.url), "utf8");
    assert.match(source, /data-settings-panel="libraries"/, `${label} no longer mount into Libraries`);
    assert.doesNotMatch(source, /data-settings-panel="metadata"/, `${label} still target the retired page`);
  }
});

test("no page mounts a panel inside another panel", () => {
  // Two pages rendered at once whenever a panel was inserted into its neighbour: the metadata sections
  // showed under Libraries, and the editor page was moved inside Modules.
  for (const file of [
    "metadata-review-template.js",
    "modules-settings.js",
    "storage-settings-template.js",
    "hardcover-shelf-settings.js",
    "metadata-provider-settings-template.js",
  ]) {
    const source = readFileSync(new URL(`../static/${file}`, import.meta.url), "utf8");
    assert.doesNotMatch(
      source,
      /insertAdjacentHTML\("beforeend"[\s\S]{0,200}data-settings-panel="[a-z]+"\]'\)/,
      `${file} inserts a panel into another panel`,
    );
    assert.doesNotMatch(source, /\.append\(editor\)/, `${file} moves a panel inside another`);
  }
});
