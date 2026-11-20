import test from "node:test";
import assert from "node:assert/strict";

test("code copy controls copy only the code text", async () => {
  let button;
  const block = {
    querySelector(selector) {
      if (selector === ":scope > .code-copy") return null;
      if (selector === "code") return { textContent: "const answer = 42" };
    },
    append(value) { button = value; },
  };
  const root = { querySelectorAll: () => [block] };
  const previousDocument = globalThis.document;
  const previousWindow = globalThis.window;
  globalThis.document = { createElement: () => ({ setAttribute() {} }) };
  globalThis.window = { setTimeout() {} };
  try {
    const { bindCodeCopy } = await import("../static/code-copy.js");
    const copied = [];
    bindCodeCopy(root, { writeText: async (value) => copied.push(value) });
    assert.ok(button);
    await button.onclick();
    assert.deepEqual(copied, ["const answer = 42"]);
    assert.equal(button.textContent, "Copied");
  } finally {
    globalThis.document = previousDocument;
    globalThis.window = previousWindow;
  }
});
