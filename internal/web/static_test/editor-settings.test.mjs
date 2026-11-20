import test from "node:test";
import assert from "node:assert/strict";
import {
  editorAbilityFeatures,
  hermesFeatures,
  hermesPanelHTML,
} from "../static/hermes-settings.js";
import { modulesPanelHTML } from "../static/modules-settings.js";

// The Settings → Editor page held one switch and is gone; the editor is a page in the menu like Hermes. What
// remains here are the Hermes-assisted editor abilities, which live in the Hermes panel.
test("the two editor tools are Hermes abilities now, with their ids and help text unchanged", () => {
  assert.deepEqual(editorAbilityFeatures.map((feature) => feature.id), ["match", "isbn"]);
  assert.deepEqual(
    editorAbilityFeatures.map((feature) => feature.label),
    ["Ask about Hardcover matches", "Ask about ISBNs"],
  );
  assert.match(editorAbilityFeatures[0].note, /^Tools, Ask Hermes about Hardcover matches:/);
  assert.match(editorAbilityFeatures[1].note, /^Tools, Ask Hermes about ISBNs:/);
  for (const feature of editorAbilityFeatures)
    assert.equal(
      hermesFeatures.find((row) => row.id === feature.id)?.label,
      feature.label,
      "listed once, in the Hermes ability list",
    );

  // The rows render inside "What she can do", with the other abilities.
  const list = hermesPanelHTML.slice(
    hermesPanelHTML.indexOf('id="hermes-features"'),
    hermesPanelHTML.indexOf("What she has installed"),
  );
  for (const feature of editorAbilityFeatures) {
    assert.match(list, new RegExp(`data-feature="${feature.id}"`));
    assert.ok(list.includes(feature.note), `help text kept for ${feature.id}`);
  }
  assert.ok(
    list.indexOf('data-feature="library_play"') < list.indexOf('data-feature="match"'),
    "the editor tools sit next to the other abilities",
  );
});

test("Modules no longer offers editor tools of its own", () => {
  assert.doesNotMatch(modulesPanelHTML, /data-feature/);
  assert.doesNotMatch(modulesPanelHTML, /Ask about (Hardcover matches|ISBNs)/);
});
