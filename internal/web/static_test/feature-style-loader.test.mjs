import assert from "node:assert/strict";
import test from "node:test";
import { loadFeatureStyles } from "../static/feature-style-loader.js";

function fakeDocument() {
  const appended = [];
  const document = {
    baseURI: "https://media.example/",
    styleSheets: [],
    createElement: () => ({ dataset: {} }),
    head: {
      append(link) {
        appended.push(link);
        document.styleSheets.push({ href: new URL(link.href, document.baseURI).href });
        queueMicrotask(() => link.onload());
      },
    },
  };
  return { document, appended };
}

test("feature styles resolve only after each stylesheet is applied", async () => {
  const { document, appended } = fakeDocument();
  let settled = false;
  const loading = loadFeatureStyles(document, ["/assets/one.css?test=one", "/assets/two.css?test=one"]).then(() => { settled = true; });
  assert.equal(settled, false);
  assert.equal(appended.length, 2);
  await loading;
  assert.equal(settled, true);
  assert.deepEqual(appended.map(link => link.rel), ["stylesheet", "stylesheet"]);
});

test("already applied route styles are not appended again", async () => {
  const { document, appended } = fakeDocument();
  document.styleSheets.push({ href: "https://media.example/assets/existing.css?v=1" });
  await loadFeatureStyles(document, ["/assets/existing.css?v=2"]);
  assert.equal(appended.length, 0);
});
