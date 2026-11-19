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
