import test from "node:test";
import assert from "node:assert/strict";
import { createShellLayout } from "../static/shell-layout.js";

test("shell layout derives one sticky offset from the unified masthead", () => {
  const values = new Map();
  const style = {
    getPropertyValue: (key) => values.get(key) || "",
    setProperty: (key, value) => values.set(key, value),
  };
  const listeners = {},
    observed = [];
  class Observer {
    constructor(listener) {
      this.listener = listener;
    }
    observe(value) {
      observed.push(value);
    }
  }
  const layout = createShellLayout({
    window: {
      scrollY: 0,
      addEventListener: (name, listener) => {
        listeners[name] = listener;
      },
    },
    root: { style },
    body: { classList: { toggle() {} } },
    masthead: { offsetHeight: 62 },
    tools: { offsetHeight: 44 },
    libraryView: { hidden: false },
    onScroll() {},
    ResizeObserver: Observer,
    requestFrame: (callback) => {
      callback();
      return 1;
    },
  });
  layout.updateStickyMetrics();
  assert.equal(values.get("--masthead-height"), "62px");
  assert.equal(values.get("--catalog-sticky-top"), "62px");
  assert.equal(observed.length, 2);
  assert.ok(listeners.scroll && listeners.resize);
});
