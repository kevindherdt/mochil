import test from "node:test";
import assert from "node:assert/strict";
import { createEditorSettings } from "../static/editor-settings.js";
import {
  editorAbilityFeatures,
  hermesFeatures,
  hermesPanelHTML,
} from "../static/hermes-settings.js";
import { modulesPanelHTML } from "../static/modules-settings.js";

// The editor section is static markup in index.html now; this module only hides
// it when the editor module is off.
function fakeDocument() {
  const editorPanel = { hidden: false };
  const document = { querySelector: (selector) => (selector === '[data-settings-panel="editor"]' ? editorPanel : null) };
  const $ = (selector) => document.querySelector(selector);
  return { document, $, editorPanel };
}

const modules = () => [{ id: "editor" }, { id: "hermes" }];
const settingsWith = (moduleState) => ({ modules: moduleState });

test("the editor section follows the editor module and nothing else", () => {
  const { $, editorPanel } = fakeDocument();
  const settingsModule = createEditorSettings({ $ });

  settingsModule.load(settingsWith({ editor: false }), modules());
  assert.equal(editorPanel.hidden, true);

  settingsModule.load(settingsWith({ editor: true }), modules());
  assert.equal(editorPanel.hidden, false);

  // Hermes has no say over the editor section any more; its abilities live in the Hermes panel.
  settingsModule.load(settingsWith({ editor: true, hermes: false }), modules());
  assert.equal(editorPanel.hidden, false);
});

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
