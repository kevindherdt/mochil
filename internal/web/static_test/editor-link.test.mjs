import assert from "node:assert/strict";
import test from "node:test";
import { createEditorMenu, editorAvailable } from "../static/editor-link.js";

const desktop = { defaultView: { matchMedia: () => ({ matches: true }) } };
const phone = { defaultView: { matchMedia: () => ({ matches: false }) } };

function fetcher({ modules, probe = 200 }) {
  return async (url) => (url === "/api/modules" ? { json: async () => modules } : { status: probe });
}

test("the editor is offered to an administrator on a desktop when the module is on", async () => {
  assert.equal(await editorAvailable(desktop, fetcher({ modules: [{ id: "editor", enabled: true }] })), true);
  assert.equal(await editorAvailable(desktop, fetcher({ modules: { data: [{ id: "editor", enabled: true }] } })), true);
  assert.equal(await editorAvailable({}, fetcher({ modules: [{ id: "editor", enabled: true }] })), true);
});

test("it is left out when the module is off, or the person is not an administrator", async () => {
  assert.equal(await editorAvailable(desktop, fetcher({ modules: [{ id: "editor", enabled: false }] })), false);
  assert.equal(await editorAvailable(desktop, fetcher({ modules: [{ id: "audio", enabled: true }] })), false);
  assert.equal(await editorAvailable(desktop, fetcher({ modules: [{ id: "editor", enabled: true }], probe: 403 })), false);
});

test("a phone or touch tablet is never offered the desktop editor, and does not even ask", async () => {
  let asked = false;
  assert.equal(await editorAvailable(phone, async () => { asked = true; return { json: async () => [{ id: "editor", enabled: true }], status: 200 }; }), false);
  assert.equal(asked, false);
});

test("a failing request never breaks the page", async () => {
  assert.equal(await editorAvailable(desktop, async () => { throw new Error("offline"); }), false);
});

test("the editor menu entry needs the server's offer and the person's choice, and remembers that choice", () => {
  const store = new Map();
  const storage = { getItem: (key) => store.get(key) ?? null, setItem: (key, value) => store.set(key, value) };
  let changes = 0;
  const menu = createEditorMenu({ storage, onChange: () => { changes += 1; } });
  assert.deepEqual(menu.entries(), []);
  menu.setAvailable(true);
  assert.deepEqual(menu.entries(), [{ id: "__editor__", name: "Library editor" }]);
  menu.setShown(false);
  assert.deepEqual(menu.entries(), []);
  assert.equal(createEditorMenu({ storage }).shown(), false);
  assert.equal(changes, 2);
});

test("choosing the Library editor opens it in this tab, like Hermes, not in a new window", () => {
  const opened = [];
  const menu = createEditorMenu({ storage: null, open: () => opened.push("/editor") });
  menu.toolbar.openEditor();
  assert.deepEqual(opened, ["/editor"]);
});
