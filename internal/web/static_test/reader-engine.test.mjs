import assert from "node:assert/strict";
import test from "node:test";
import { engineKey, mountReaderEngine, readerEngine } from "../static/reader-engine.js";

const memory = (initial = {}) => {
  const values = { ...initial };
  return { getItem: (key) => values[key] ?? null, setItem: (key, value) => (values[key] = String(value)), values };
};

test("Readium is the default and unknown values fall back to it", () => {
  assert.equal(readerEngine(memory()), "readium");
  assert.equal(readerEngine(memory({ [engineKey]: "mystery" })), "readium");
  assert.equal(readerEngine({ getItem() { throw new Error("blocked"); } }), "readium");
});

test("a saved choice applies and the address can override it once", () => {
  const storage = memory({ [engineKey]: "classic" });
  assert.equal(readerEngine(storage), "classic");
  assert.equal(readerEngine(storage, "?engine=readium"), "readium");
  assert.equal(readerEngine(memory(), "?engine=classic"), "classic");
});

test("Settings keeps only the read-along colour; the engine picker is hidden", () => {
  const storage = memory({ [engineKey]: "classic" });
  let color;
  const queried = [];
  const card = {
    querySelector: (selector) => { queried.push(selector); return selector === "#read-along-color" ? color : null; },
    insertAdjacentHTML(_, html) {
      assert.doesNotMatch(html, /reader-engine|Book reader|Readium|Classic/);
      color = { value: "", listener: null, addEventListener(type, fn) { this.listener = fn; } };
    },
  };
  mountReaderEngine({ querySelector: () => card }, storage);
  assert.equal(color.value, "amber");
  color.value = "blue";
  color.listener();
  assert.equal(storage.values["bobobooks-read-along-color"], "blue");
  assert.equal(storage.values[engineKey], "classic", "a saved engine choice is left alone");
});

async function redirect(search, saved) {
  const { readFile } = await import("node:fs/promises");
  const { runInNewContext } = await import("node:vm");
  const source = await readFile(new URL("../static/reader-engine-redirect.js", import.meta.url), "utf8");
  const calls = [];
  const location = { search, pathname: "/reader.html", hash: "#x", replace: (url) => calls.push(url) };
  const localStorage = { getItem: () => saved ?? null };
  runInNewContext(source, { location, localStorage, URLSearchParams });
  return calls;
}

test("the escape hatch still works: ?engine=classic or a saved classic stays on the classic reader, everything else goes to Readium", async () => {
  assert.deepEqual(await redirect("?engine=classic"), []);
  assert.deepEqual(await redirect("", "classic"), []);
  assert.deepEqual(await redirect(""), ["/reader.html?engine=readium#x"]);
});
