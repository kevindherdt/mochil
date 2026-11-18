import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

function load(fetchImpl = async () => ({ ok: true, clone() { return this; } })) {
  const listeners = {};
  const store = new Map();
  const cacheOf = (name) => {
    if (!store.has(name)) store.set(name, new Map());
    const entries = store.get(name);
    return {
      match: async (request) => entries.get(typeof request === "string" ? request : request.url),
      put: async (request, response) => { entries.set(typeof request === "string" ? request : request.url, response); },
      keys: async () => [...entries.keys()].map((url) => ({ url })),
      delete: async (key) => entries.delete(key.url),
      addAll: async () => {},
    };
  };
  const caches = { open: async (name) => cacheOf(name), keys: async () => [...store.keys()], delete: async (name) => store.delete(name) };
  const self = { addEventListener: (type, fn) => { listeners[type] = fn; }, location: { origin: "https://books.test" }, skipWaiting() {}, clients: { claim: async () => {} } };
  vm.runInNewContext(readFileSync(new URL("../static/sw.js", import.meta.url), "utf8"), { self, caches, URL, Response, AbortController, fetch: fetchImpl, setTimeout, clearTimeout });
  return { listeners, store, cacheOf };
}

test("slow navigation waits for the server rather than producing an offline page", async () => {
  let finish;
  const { listeners } = load(() => new Promise((resolve) => { finish = resolve; }));
  let answer;
  listeners.fetch({ request: { method: "GET", url: "https://books.test/signin", mode: "navigate" }, respondWith: (p) => { answer = p; } });
  const response = new Response("sign in");
  await new Promise((resolve) => setTimeout(resolve, 2100));
  finish(response);
  assert.equal(await answer, response);
});

test("network failure serves the cached shell only for the root route", async () => {
  const { listeners, store, cacheOf } = load(async () => { throw new Error("offline"); });
  cacheOf("bobobooks-shell-v1");
  const shell = new Response("shell");
  store.get("bobobooks-shell-v1").set("/", shell);
  async function navigate(path) {
    let answer;
    listeners.fetch({ request: { method: "GET", url: `https://books.test${path}`, mode: "navigate" }, respondWith: p => { answer = p; } });
    return answer;
  }
  assert.equal(await navigate("/"), shell);
  assert.equal((await navigate("/signin")).status, 503);
});

test("server authentication responses pass through unchanged", async () => {
  const response = new Response("sign in", { status: 401 });
  const { listeners } = load(async () => response);
  let answer;
  listeners.fetch({ request: { method: "GET", url: "https://books.test/", mode: "navigate" }, respondWith: p => { answer = p; } });
  assert.equal(await answer, response);
});

test("activation removes only Mochil caches that are no longer current", async () => {
  const { listeners, store, cacheOf } = load();
  for (const name of ["bobobooks-shell-v1", "bobobooks-assets-v1", "bobobooks-assets-v0", "another-app-cache"]) cacheOf(name);
  let done;
  listeners.activate({ waitUntil: (promise) => { done = promise; } });
  await done;
  assert.deepEqual([...store.keys()].sort(), ["another-app-cache", "bobobooks-assets-v1", "bobobooks-shell-v1"]);
});

test("the asset cache stays bounded and keeps the newest bundles", async () => {
  const { listeners, store } = load();
  const waits = [];
  for (let i = 0; i < 320; i += 1) {
    let answered;
    listeners.fetch({ request: { method: "GET", url: `https://books.test/assets/dist/app-${i}.js`, mode: "no-cors" }, respondWith: (promise) => { answered = promise; }, waitUntil: (promise) => waits.push(promise) });
    await answered;
  }
  await Promise.all(waits);
  const urls = [...store.get("bobobooks-assets-v1").keys()];
  assert.equal(urls.length, 300);
  assert.ok(urls.at(-1).endsWith("app-319.js"));
  assert.ok(!urls.some((url) => url.endsWith("app-0.js")));
});
