import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import {
  createModulesSettings,
  effectiveModuleState,
  groupedModules,
  moduleLibraryCount,
  moduleRowHTML,
  moduleRowsHTML,
} from "../static/modules-settings.js";

const video = { id: "video", name: "Video", description: "Movies and TV.", kinds: ["movies", "tv", "anime"] };
const audio = { id: "audio", name: "Audio", description: "Music.", kinds: ["music"] };
const hardcover = { id: "hardcover", name: "Hardcover shelf sync", description: "Mark books as currently reading or read on your Hardcover account." };
const progress = { id: "hardcover_progress", name: "Hardcover reading progress", description: "Also send how far you are through each book.", group: "hardcover", option: true };
const ratings = { id: "hardcover_ratings", name: "Hardcover ratings", description: "Send the star rating you give a book.", group: "hardcover", option: true };

test("a module follows its libraries until the user decides", () => {
  const books = { libraries: [{ kind: "books" }] };
  assert.equal(effectiveModuleState(video, books), false);
  assert.equal(effectiveModuleState(video, { libraries: [{ kind: "tv" }] }), true);
  assert.equal(effectiveModuleState(video, { libraries: [{ kind: "tv" }], modules: { video: false } }), false);
  assert.equal(effectiveModuleState(audio, { libraries: [], modules: { audio: true } }), true);
  assert.equal(moduleLibraryCount(video, { libraries: [{ kind: "tv" }, { kind: "movies" }, { kind: "books" }] }), 2);
});

test("the server's own enabled field wins over the client's recomputed guess", () => {
  // GET /api/modules already computes enabled via the backend's authoritative
  // moduleOn(); the settings-derived heuristic must never override it, even when
  // settings.libraries/modules would otherwise disagree (e.g. a library added
  // after /api/settings was cached, or a module forced on server-side).
  const serverSaysOn = { ...video, enabled: true };
  assert.equal(effectiveModuleState(serverSaysOn, { libraries: [] }), true);
  const serverSaysOff = { ...audio, enabled: false };
  assert.equal(effectiveModuleState(serverSaysOff, { libraries: [{ kind: "music" }], modules: { audio: true } }), false);
});

test("rows say what is affected and expose an accessible switch", () => {
  const off = moduleRowHTML(video, false, 2);
  assert.match(off, /2 libraries hidden while off/);
  assert.match(off, /role="switch" aria-checked="false"/);
  assert.match(moduleRowHTML(video, true, 1), /1 library</);
  assert.match(moduleRowHTML(video, false, 0), /No libraries use it yet/);
  assert.doesNotMatch(moduleRowHTML({ ...video, name: "<b>x" }, true, 0), /<b>x/);
});

test("a module marked as an option joins its group and keeps the server order", () => {
  assert.deepEqual(
    groupedModules([video, hardcover, progress, ratings, audio]).map(({ module, children }) => [module.id, children.map((child) => child.id)]),
    [["video", []], ["hardcover", ["hardcover_progress", "hardcover_ratings"]], ["audio", []]],
  );
  // An option whose card was not sent must still be reachable.
  assert.deepEqual(groupedModules([progress]).map(({ module }) => module.id), ["hardcover_progress"]);
  // A card never becomes an option of itself.
  assert.deepEqual(
    groupedModules([{ ...hardcover, group: "hardcover", option: true }, progress]).map(({ module, children }) => [module.id, children.map((child) => child.id)]),
    [["hardcover", ["hardcover_progress"]]],
  );
});

test("the Hardcover card keeps saying what shelf sync does", () => {
  const html = moduleRowsHTML(
    [{ ...hardcover, description: "Account link." }, progress],
    { hardcover: true, hardcover_progress: true },
    {},
  );
  assert.match(html, /Shelf sync: Account link\./);
  assert.match(moduleRowsHTML([hardcover, progress], { hardcover: true }, {}), /reading or read on your Hardcover account/);
});

test("one Hardcover card holds its options, with the ungrouped rows untouched", () => {
  const state = { video: true, hardcover: true, hardcover_progress: true, hardcover_ratings: false, audio: false };
  const html = moduleRowsHTML([video, hardcover, progress, ratings, audio], state, { libraries: [{ kind: "tv" }] });
  assert.match(html, /class="module-row module-group"/);
  assert.match(html, /<strong>Hardcover<em class="module-pill" data-state="on">Running<\/em><\/strong>/);
  assert.match(html, /class="module-row module-sub"/);
  assert.match(html, /data-parent="hardcover"/);
  assert.doesNotMatch(html, /<strong>Hardcover shelf sync/);
  assert.ok(html.indexOf('data-module="video"') < html.indexOf("module-row module-group"));
  assert.ok(html.indexOf("module-row module-group") < html.indexOf('data-module="audio"'));
  // The plain rows are exactly what they were before grouping existed.
  assert.ok(html.includes(moduleRowHTML(video, true, 1)));
  assert.ok(html.includes(moduleRowHTML(audio, false, 0)));
  // Two options, one card: no duplicate parent rows.
  assert.equal(html.match(/data-module="hardcover"/g).length, 1);
});

test("switching the card off disables its options without clearing them", () => {
  const settings = { libraries: [], modules: { hardcover: true, hardcover_progress: true, hardcover_ratings: true } };
  const { modules, list } = fixture(settings, [hardcover, progress, ratings]);
  assert.match(list.innerHTML, /aria-checked="true" aria-label="Hardcover reading progress" data-module="hardcover_progress"/);

  list.listener({ target: { closest: () => ({ dataset: { module: "hardcover" } }) } });
  const off = list.innerHTML;
  assert.match(off, /class="module-row module-sub" data-disabled="true"/);
  // An option that cannot run must not announce itself as on: the switch reads
  // off and its label says the choice is saved rather than active.
  assert.match(off, /aria-checked="false"[^>]*aria-label="Hardcover reading progress, saved but off until its module is on"[^>]*data-module="hardcover_progress"[^>]*disabled/);

  // The saved state survives: only the switch the user moved becomes a setting.
  const target = { modules: { ...settings.modules } };
  modules.write(target);
  assert.deepEqual(target.modules, { ...settings.modules, hardcover: false });

  // Clicking a disabled option changes nothing.
  list.listener({ target: { closest: () => ({ dataset: { module: "hardcover_progress" }, disabled: true }) } });
  assert.equal(list.innerHTML, off);

  list.listener({ target: { closest: () => ({ dataset: { module: "hardcover" } }) } });
  assert.doesNotMatch(list.innerHTML, /data-disabled="true"/);
  assert.match(list.innerHTML, /aria-checked="true"[^>]*data-module="hardcover_progress"/);
});

test("only switches the user moved become explicit settings", () => {
  const settings = { libraries: [{ kind: "tv" }], modules: { audio: false } };
  const { modules, list, dirtyCount } = fixture(settings);
  const target = JSON.parse(JSON.stringify(settings));
  modules.write(target);
  assert.deepEqual(target.modules, { audio: false }, "nothing touched, nothing changed");
  list.listener({ target: { closest: () => ({ dataset: { module: "video" } }) } });
  list.listener({ target: { closest: () => ({ dataset: { module: "audio" } }) } });
  assert.equal(dirtyCount(), 2);
  modules.write(target);
  assert.deepEqual(target.modules, { audio: true, video: false });
});

test("the Editor page stays its own page, and follows the editor switch", () => {
  // It used to be moved inside the Modules panel, so it showed on the Modules page and vanished from its
  // own. Its visibility follows the module switch; its home is the settings view like every other page.
  const editor = { hidden: true, parent: null };
  const panel = { appended: [], append(node) { this.appended.push(node); node.parent = this; } };
  const view = { insertAdjacentHTML() { panel.mounted = true; } };
  const list = { innerHTML: "", listener: null, addEventListener(_t, fn) { this.listener = fn; } };
  const document = {
    querySelector: (selector) => selector === '[data-settings-panel="modules"]' ? (panel.mounted ? panel : null) : selector === '[data-settings-panel="editor"]' ? editor : view,
  };
  const elements = new Map([["#module-list", list], ['[data-settings-panel="editor"]', editor]]);
  const $ = (selector) => elements.get(selector) || { focus() {} };
  const modules = createModulesSettings({ document, $, onDirty() {} });
  assert.deepEqual(panel.appended, [], "the editor page was nested inside Modules again");
  modules.bind();
  modules.load([{ id: "editor", name: "Editor", kinds: [] }, video], { modules: { editor: false } });
  assert.equal(editor.hidden, true);
  list.listener({ target: { closest: () => ({ dataset: { module: "editor" } }) } });
  assert.equal(editor.hidden, false, "switching the editor on shows its page at once");
});

function fixture(settings, modules = [video, audio]) {
  const elements = new Map();
  const list = { innerHTML: "", listener: null, addEventListener(_t, fn) { this.listener = fn; } };
  elements.set("#module-list", list);
  const document = { querySelector: () => ({}), };
  let dirty = 0;
  const $ = (selector) => elements.get(selector) || { focus() {} };
  const controller = createModulesSettings({ document, $, onDirty: () => dirty++ });
  controller.bind();
  controller.load(modules, settings);
  return { modules: controller, list, dirtyCount: () => dirty };
}

test('a module change asks for a restart once, not on every later save', () => {
  // Toggling narration off and on asked for a restart, and then every subsequent save in the same
  // visit asked again -- the moved switches were never cleared after the save that wrote them. Any
  // settings save after that looked like a module change, over a change that was already applied.
  const source = readFileSync(new URL('../static/modules-settings.js', import.meta.url), 'utf8')
  assert.match(source, /saved\(\)/, 'the module panel must expose a way to record that it was saved')
  assert.match(source, /touched\.clear\(\);\n\s*\/\/ The switches just written/, 'saving clears the moved set')
  const settings = readFileSync(new URL('../static/settings.js', import.meta.url), 'utf8')
  assert.match(settings, /modulesSettings\.saved\(\)/, 'the save path must call it after a successful write')
  assert.ok(
    settings.indexOf('modulesSettings.saved()') < settings.indexOf('if (modulesChanged) offerRestart'),
    'the set is cleared before the prompt is offered for it',
  )
})
