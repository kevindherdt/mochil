import test from "node:test";
import assert from "node:assert/strict";
import { createCatalogueToolbar } from "../static/catalogue-toolbar.js";

function element() {
  return {
    classList: { toggle() {} },
    setAttribute() {},
    closest() {
      return this.label || (this.label = {});
    },
    focus() {
      this.focused = true;
    },
  };
}

function layoutButtons() {
  return ["covers", "list"].map((layout) => ({
    dataset: { layout },
    pressed: "",
    setAttribute(name, value) {
      if (name === "aria-pressed") this.pressed = value;
    },
  }));
}

function fixture({ vault = false, video = false, audio = false, documents = false, capturePlace, restorePlace } = {}) {
  const selectors = [
    "#filters",
    "#view-menu",
    ".browse-controls",
    "#video-layout",
    "#document-layout",
    "#search",
    "#search-panel .sr-only",
    "#series-layout",
    "#item-layout",
    ".audio-browse-head",
  ];
  const elements = Object.fromEntries(
    selectors.map((name) => [name, element()]),
  );
  const buttons = layoutButtons();
  elements["#view-menu"].querySelectorAll = () => buttons;
  elements["#video-layout"].value = "covers";
  elements["#document-layout"].value = "covers";
  const state = {
    selected: "books",
    collection: false,
    browse: "series",
    seriesLayout: "covers",
    itemLayout: "covers",
  };
  const saved = {};
  const calls = [];
  const toolbar = createCatalogueToolbar({
    $: (selector) => elements[selector],
    escapeHTML: String,
    storage: { setItem: (key, value) => (saved[key] = value) },
    libraryState: {
      visible: () => [{ id: "books", name: "Books", count: 3 }],
      all: () => [{ id: "books" }],
      selected: () => state.selected,
      select: (value) => {
        state.selected = value;
        saved["bobobooks-library"] = value;
      },
    },
    getCollectionMode: () => state.collection,
    setCollectionMode: (value) => (state.collection = value),
    getBrowseMode: () => state.browse,
    getSeriesLayout: () => state.seriesLayout,
    setSeriesLayout: (value) => (state.seriesLayout = value),
    getItemLayout: () => state.itemLayout,
    setItemLayout: (value) => (state.itemLayout = value),
    isVideoLibrary: () => video,
    isAudioLibrary: () => audio,
    isDocumentLibrary: () => documents,
    isVaultLibrary: () => vault,
    renderAlphabet: () => calls.push("alphabet"),
    loadCurrentView: () => calls.push("load"),
    reflowCollection: () => (state.openCollection ? (calls.push("reflow"), true) : false),
    capturePlace: capturePlace || (() => null),
    restorePlace: restorePlace || (() => {}),
    openLibrary: () => calls.push("load"),
    openHome: () => calls.push("home"),
  });
  toolbar.bind();
  return { buttons, calls, elements, saved, state, toolbar };
}

test("library selection clears collections, persists, and reloads", () => {
  const { calls, elements, saved, state, toolbar } = fixture();
  toolbar.render();
  state.collection = true;
  elements["#filters"].onchange({
    target: { ...elements["#filters"], value: "manga" },
  });
  assert.equal(state.selected, "manga");
  assert.equal(state.collection, false);
  assert.equal(saved["bobobooks-library"], "manga");
  assert.equal(calls.at(-1), "load");
});

test("one layout switch sets groups and books together, persists, and reloads once", () => {
  const { buttons, calls, saved, state, toolbar } = fixture();
  state.browse = "authors";
  toolbar.render();
  calls.length = 0;
  buttons[1].onclick();
  assert.equal(state.seriesLayout, "list");
  assert.equal(state.itemLayout, "list");
  assert.equal(saved["bobobooks-series-layout"], "list");
  assert.equal(saved["bobobooks-item-layout"], "list");
  assert.deepEqual(calls, ["load"]);
  assert.deepEqual(buttons.map((button) => button.pressed), ["false", "true"]);
});

test("the switch stays available in collections", () => {
  const { buttons, elements, state, toolbar } = fixture();
  state.collection = true;
  toolbar.render();
  assert.equal(elements["#view-menu"].hidden, false);
  buttons[1].onclick();
  assert.equal(state.itemLayout, "list");
});

test("video and document shelves drive their own layout selects", () => {
  for (const [options, id] of [[{ video: true }, "#video-layout"], [{ documents: true }, "#document-layout"]]) {
    const { buttons, calls, elements, state } = fixture(options);
    let changed;
    elements[id].onchange = () => (changed = elements[id].value);
    calls.length = 0;
    buttons[1].onclick();
    assert.equal(changed, "list");
    assert.equal(state.itemLayout, "covers");
    assert.deepEqual(calls, []);
  }
});

test("Home is part of the destination selector without becoming a library",()=>{
  const {calls,elements,state,toolbar}=fixture()
  toolbar.render()
  assert.match(elements["#filters"].innerHTML,/value="__home__" data-kind="home">Home/)
  assert.match(elements["#filters"].innerHTML,/value="__settings__" data-kind="settings">Settings/)
  assert.doesNotMatch(elements["#filters"].innerHTML,/All shelves/)
  elements["#filters"].onchange({target:{...elements["#filters"],value:"__home__"}})
  assert.equal(state.selected,"books")
  assert.equal(calls.at(-1),"home")
})

test("Wiki vaults hide the irrelevant cover and list control", () => {
  const { elements, toolbar } = fixture({ vault: true });
  toolbar.render();
  assert.equal(elements["#view-menu"].hidden, true);
});

test("video shelves keep the layout switch and retire stale audio navigation", () => {
  const { elements, toolbar } = fixture({ video: true });
  elements[".audio-browse-head"].hidden = false;
  toolbar.render();
  assert.equal(elements["#view-menu"].hidden, false);
  assert.equal(elements[".audio-browse-head"].hidden, true);
});

test("switching layout inside an opened collection redraws it instead of reloading the collections list", () => {
  const { buttons, calls, state, toolbar } = fixture();
  state.collection = true;
  state.openCollection = true;
  toolbar.render();
  calls.length = 0;
  buttons[1].onclick();
  assert.deepEqual(calls, ["reflow"]);
  state.openCollection = false;
  calls.length = 0;
  buttons[0].onclick();
  assert.deepEqual(calls, ["load"]);
});

// Switching Covers ⇄ List reloads the shelf and used to drop the person at the top of it
// (scrollY 2600 → 0). The switch carries the visible card across the reload.
test("a layout switch carries the visible card across the reload", () => {
  const events = [];
  const place = { container: "books", key: "book:lib/b", top: 120 };
  const { buttons, state, toolbar } = fixture({
    capturePlace: () => (events.push("capture"), place),
    restorePlace: (value) => events.push(["restore", value]),
  });
  state.browse = "books";
  toolbar.render();
  events.length = 0;
  buttons[1].onclick();
  assert.equal(events[0], "capture", "the old layout is measured before it is replaced");
  assert.deepEqual(events[1], ["restore", place]);
  assert.equal(state.itemLayout, "list");
});
