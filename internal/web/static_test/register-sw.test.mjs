import assert from "node:assert/strict";
import test from "node:test";
import { registerServiceWorker } from "../static/register-sw.js";

test("registers the worker after load, scoped to the whole app", async () => {
  const calls = [];
  let onLoad;
  const win = {
    navigator: { serviceWorker: { register: (...args) => { calls.push(args); return Promise.resolve(); } } },
    addEventListener: (type, fn) => { if (type === "load") onLoad = fn; },
  };
  registerServiceWorker(win);
  assert.equal(calls.length, 0, "nothing runs before the page has loaded");
  onLoad();
  assert.deepEqual(calls, [["/sw.js", { scope: "/" }]]);
});

test("browsers without service workers and failed registrations are harmless", async () => {
  registerServiceWorker({ navigator: {}, addEventListener: () => assert.fail("must not listen") });
  let onLoad;
  registerServiceWorker({
    navigator: { serviceWorker: { register: () => Promise.reject(new Error("blocked")) } },
    addEventListener: (_t, fn) => (onLoad = fn),
  });
  await onLoad();
});

// The page must never repaint itself because a worker took over: a first install claims the page it is
// running in, and a new build replaces a live worker, and both fire controllerchange.
function fixture() {
  const reloads = [];
  const registrations = [];
  const listeners = {};
  const win = {
    navigator: { serviceWorker: { controller: {}, register: (...args) => { registrations.push(args); return Promise.resolve(); }, addEventListener() {} } },
    location: { reload: () => reloads.push(1) },
    addEventListener: (type, fn) => { listeners[`win:${type}`] = fn; },
    setInterval: () => 0,
    clearInterval: () => {},
    document: { querySelectorAll: () => [], addEventListener() {}, visibilityState: "visible", createElement: () => ({ style: {} }) },
    fetch: async () => ({ ok: false }),
  };
  registerServiceWorker(win);
  listeners["win:load"]();
  return { reloads, registrations };
}

test("a worker taking over never reloads the page underneath the reader", () => {
  const { reloads, registrations } = fixture();
  assert.equal(registrations.length, 1, "the worker is still registered on load");
  assert.deepEqual(reloads, [], "a new build is offered with the update bar, not forced");
});
