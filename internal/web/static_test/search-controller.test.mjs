import test from "node:test";
import assert from "node:assert/strict";
import { createSearchController } from "../static/search-controller.js";

function control() {
  return {
    attributes: {},
    setAttribute(name, value) {
      this.attributes[name] = String(value);
    },
    getAttribute(name) {
      return this.attributes[name];
    },
    focus() {
      this.focused = true;
    },
  };
}

test("search controller opens, focuses, and closes on Escape", () => {
  const toggle = control(),
    panel = control(),
    input = control();
  let keydown;
  input.addEventListener = (_type, listener) => {
    keydown = listener;
  };
  const masthead = {
    classList: {
      toggle(_name, value) {
        masthead.searching = value;
      },
    },
  };
  createSearchController({
    masthead,
    panel,
    toggle,
    input,
    label: () => "Search documents",
    onSearch() {},
    requestFrame: (callback) => callback(),
  });
  toggle.onclick();
  assert.equal(masthead.searching, true);
  assert.equal(input.focused, true);
  let prevented = false;
  keydown({
    key: "Escape",
    preventDefault() {
      prevented = true;
    },
  });
  assert.equal(prevented, true);
  assert.equal(panel.inert, true);
  assert.equal(toggle.focused, true);
  assert.equal(toggle.attributes["aria-label"], "Search documents");
});

test("Enter commits the current query and restores the library navigation", () => {
  const toggle = control(), panel = control(), input = control();
  let keydown, searches = 0;
  input.addEventListener = (_type, listener) => { keydown = listener; };
  const masthead = { classList: { toggle(_name, value) { masthead.searching = value; } } };
  createSearchController({ masthead, panel, toggle, input, label: () => "Search music", onSearch: () => { searches++; }, requestFrame: callback => callback() });
  toggle.onclick();
  let prevented = false;
  keydown({ key: "Enter", isComposing: false, preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.equal(searches, 1);
  assert.equal(masthead.searching, false);
  assert.equal(panel.inert, true);
  assert.equal(toggle.attributes["aria-expanded"], "false");
});

const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

function searchFixture({ delay = 10 } = {}) {
  const toggle = control(), panel = control(), input = control();
  input.value = "";
  let keydown, searches = 0;
  input.addEventListener = (_type, listener) => { keydown = listener; };
  const masthead = { classList: { toggle(_name, value) { masthead.searching = value; } } };
  createSearchController({ masthead, panel, toggle, input, label: () => "Search books", onSearch: () => { searches++; }, requestFrame: callback => callback(), delay });
  return { toggle, panel, input, masthead, keydown: () => keydown, searches: () => searches };
}

// Typing and dismissing inside the debounce window used to leave the timer running, so the shelf was
// replaced by results for a query the closed panel no longer showed.
test("dismissing the panel cancels the pending search", async () => {
  const search = searchFixture();
  search.toggle.onclick();
  search.input.value = "dune";
  search.input.oninput();
  search.toggle.onclick();
  assert.equal(search.masthead.searching, false);
  await pause(30);
  assert.equal(search.searches(), 0, "the debounce must not outlive the panel");
});

test("Escape closes, cancels the pending search, and restores the unfiltered shelf", async () => {
  const search = searchFixture();
  search.toggle.onclick();
  search.input.value = "dune";
  search.input.oninput();
  await pause(30);
  assert.equal(search.searches(), 1, "the applied query reached the shelf");
  search.keydown()({ key: "Escape", preventDefault() {} });
  assert.equal(search.input.value, "");
  assert.equal(search.searches(), 2, "closing put the unfiltered shelf back");
  assert.equal(search.panel.inert, true);
});

test("submitting with Enter keeps the query and does not undo it", () => {
  const search = searchFixture();
  search.toggle.onclick();
  search.input.value = "dune";
  search.keydown()({ key: "Enter", isComposing: false, preventDefault() {} });
  assert.equal(search.searches(), 1);
  assert.equal(search.input.value, "dune");
  assert.equal(search.masthead.searching, false);
});
