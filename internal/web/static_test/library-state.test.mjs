import test from "node:test";
import assert from "node:assert/strict";
import { createLibraryState } from "../static/library-state.js";

function fixture(saved = null) {
  const values = saved === null ? {} : { "mochil-library": saved };
  const state = createLibraryState({
    storage: {
      getItem: (key) => values[key] ?? null,
      setItem: (key, value) => (values[key] = value),
    },
  });
  state.setAll([{ id: "books" }, { id: "manga" }]);
  state.setVisible([{ id: "books" }, { id: "manga" }]);
  return { state, values };
}

test("initial selection restores a visible saved shelf", () => {
  const { state } = fixture("manga");
  assert.equal(state.initializeSelection(), "manga");
  assert.equal(state.selectedRecord().id, "manga");
});

test("initial selection rejects a hidden or unknown shelf", () => {
  const { state, values } = fixture("missing");
  assert.equal(state.initializeSelection(), "books");
  assert.equal(values["mochil-library"], "books");
});

test("an old aggregate selection migrates to the first visible shelf", () => {
  const { state } = fixture("");
  assert.equal(state.initializeSelection(), "books");
  assert.equal(state.ensureVisible(), "books");
  state.select("manga");
  state.setVisible([{ id: "books" }]);
  assert.equal(state.ensureVisible(), "books");
});
