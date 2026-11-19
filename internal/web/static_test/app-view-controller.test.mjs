import test from "node:test";
import assert from "node:assert/strict";
import { createAppViewController } from "../static/app-view-controller.js";

function element() {
  return {
    hidden: false,
    attributes: {},
    classList: { toggle() {} },
    setAttribute(name, value) {
      this.attributes[name] = String(value);
    },
    getAttribute(name) {
      return this.attributes[name];
    },
    removeAttribute(name) {
      delete this.attributes[name];
    },
  };
}

function fixture() {
  const selectors = [
    '[data-view="settings"]',
    '[data-view="sources"]',
    "#search-panel",
    "#dashboard-view",
    "#library-view",
    "#source-view",
    "#settings-view",
    "#shell-context",
  ];
  const elements = Object.fromEntries(
    selectors.map((name) => [name, element()]),
  );
  const calls = [];
  const controller = createAppViewController({
    $: (selector) => elements[selector],
    mediaNavigator: { hide: () => calls.push("hide-navigation") },
    searchController: { setOpen: (open) => calls.push(`search:${open}`) },
    searchToggle: element(),
    hasLibraries: () => true,
    openSettings: () => calls.push("settings"),
    loadLibrary: () => calls.push("library"),
    loadSources: () => calls.push("sources"),
    loadDashboard: () => calls.push("dashboard"),
    updateLayout: () => calls.push("layout"),
    requestFrame: (callback) => callback(),
  });
  return { calls, controller, elements };
}

test("view controller exposes one active surface and invokes its loader", () => {
  const { calls, controller, elements } = fixture();
  controller.show("settings");
  assert.equal(controller.current(), "settings");
  assert.equal(elements["#settings-view"].hidden, false);
  assert.equal(elements["#dashboard-view"].hidden, true);
  assert.equal(
    elements['[data-view="settings"]'].attributes["aria-pressed"],
    "true",
  );
  assert.deepEqual(calls, [
    "hide-navigation",
    "search:false",
    "settings",
    "layout",
  ]);
});

test("clicking the active settings button returns to dashboard", () => {
  const { calls, controller, elements } = fixture();
  controller.bind();
  controller.show("settings");
  elements['[data-view="settings"]'].onclick();
  assert.equal(controller.current(), "dashboard");
  assert.equal(calls.includes("settings"), true);
  assert.equal(calls.at(-2), "dashboard");
  assert.equal(elements['[data-view="settings"]'].attributes["aria-pressed"], "false");
});
