import test from "node:test";
import assert from "node:assert/strict";
import { createLibraryLoader } from "../static/library-loader.js";

function fixture(savedLibrary = null, libraries = [{ id: "books" }, { id: "manga" }], options = {}) {
  const state = {
    libraries: [],
    all: [],
    selected: "",
    ready: false,
  };
  const saved = { "mochil-library": savedLibrary };
  const calls = [];
  const loader = createLibraryLoader({
    requestJSON: async () => ({
      data: libraries,
    }),
    libraryVisibility: { setLibraries: (libraries) => libraries },
    readingDashboard: { load: () => calls.push("dashboard") },
    videoLibrary: { loadContinue: () => calls.push("video") },
    audioLibrary: { loadContinue: () => calls.push("audio") },
    libraryState: {
      all: () => state.all,
      setAll: (value) => (state.all = value),
      setVisible: (value) => (state.libraries = value),
      initializeSelection: () => {
        state.selected = savedLibrary || state.libraries[0]?.id || "";
        state.ready = true;
      },
      ensureVisible: () => {
        if (!state.libraries.some((item) => item.id === state.selected)) {
          state.selected = state.libraries[0]?.id || "";
          saved["mochil-library"] = state.selected;
        }
      },
    },
    renderToolbar: () => calls.push("toolbar"),
    renderAlphabet: () => calls.push("alphabet"),
    isLibraryView: () => true,
    loadCurrentView: () => calls.push("library"),
    setStatus: (message) => calls.push(message),
    onFirstRun: () => calls.push("first-run"),
    ...options,
  });
  return { calls, loader, saved, state };
}

test("visible recovery continues at the capped interval until the server returns", async () => {
  let attempts = 0;
  const { loader, state } = fixture(null, undefined, {
    retryDelays: [5, 10],
    requestJSON: async () => { if (++attempts < 5) throw new Error("unreachable"); return { data: [{ id: "books" }] }; },
  });
  await loader.load();
  await new Promise((resolve) => setTimeout(resolve, 100));
  assert.equal(attempts, 5);
  assert.equal(state.selected, "books");
  loader.dispose();
});

test("a pending retry pauses when hidden and resumes when visible", async () => {
  const doc = new EventTarget(); doc.hidden = false;
  let attempts = 0;
  const { loader } = fixture(null, undefined, { document: doc, retryDelays: [20], requestJSON: async () => { attempts++; throw new Error("offline"); } });
  await loader.load();
  doc.hidden = true; doc.dispatchEvent(new Event("visibilitychange"));
  await new Promise((resolve) => setTimeout(resolve, 60));
  assert.equal(attempts, 1);
  doc.hidden = false; doc.dispatchEvent(new Event("visibilitychange"));
  await new Promise((resolve) => setTimeout(resolve, 5));
  assert.equal(attempts, 2);
  loader.dispose();
});

test("initial route restoration that loads a library does not load it twice", async () => {
  const calls = [];
  const { loader } = fixture(null, undefined, {
    isLibraryView: () => true,
    onReady: () => { calls.push("library"); return true; },
    loadCurrentView: () => calls.push("library"),
  });
  await loader.load();
  assert.deepEqual(calls, ["library"]);
  loader.dispose();
});

test("disposing a loader ignores an in-flight response", async () => {
  let finish;
  const { loader, state } = fixture(null, undefined, { requestJSON: () => new Promise((resolve) => { finish = resolve; }) });
  const loading = loader.load();
  loader.dispose();
  finish({ data: [{ id: "books" }] });
  await loading;
  assert.equal(state.ready, false);
});

test("a settings refresh during discovery performs a fresh discovery", async () => {
  const pending = [];
  const { loader, state } = fixture(null, undefined, {
    requestJSON: () => new Promise((resolve) => pending.push(resolve)),
  });
  const first = loader.load();
  const refresh = loader.load();
  pending.shift()({ data: [{ id: "old" }] });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(pending.length, 1, "a second discovery must start");
  pending.shift()({ data: [{ id: "new" }] });
  await Promise.all([first, refresh]);
  assert.deepEqual(state.all, [{ id: "new" }]);
  loader.dispose();
});

test("a queued refresh cancels the retry scheduled by the failed first request", async () => {
  let rejectFirst, attempts = 0;
  const { loader } = fixture(null, undefined, {
    retryDelays: [15],
    requestJSON: () => {
      attempts++;
      return attempts === 1 ? new Promise((_, reject) => { rejectFirst = reject; }) : Promise.resolve({ data: [{ id: "books" }] });
    },
  });
  const first = loader.load();
  loader.load();
  rejectFirst(new Error("offline"));
  await first;
  await new Promise((resolve) => setTimeout(resolve, 45));
  assert.equal(attempts, 2);
  loader.dispose();
});

test("failed discovery does not start auxiliary dashboard requests", async () => {
  const { loader, calls } = fixture(null, undefined, {
    retryDelays: [], requestJSON: async () => { throw new Error("offline"); },
  });
  await loader.load();
  assert.equal(calls.includes("dashboard"), false);
  assert.equal(calls.includes("video"), false);
  assert.equal(calls.includes("audio"), false);
  loader.dispose();
});

test("repeated refreshes do not overlap a stalled auxiliary request", async () => {
  let calls = 0, finish;
  const { loader } = fixture(null, undefined, {
    readingDashboard: { load: () => { calls++; return new Promise((resolve) => { finish = resolve; }); } },
  });
  await loader.load();
  await loader.load();
  loader.applyVisible([{ id: "books" }]);
  assert.equal(calls, 1);
  finish();
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(calls, 2, "a refresh is coalesced after the first finishes");
  finish();
  loader.dispose();
});

test("authentication failures do not retry automatically but permit manual retry", async () => {
  const win = new EventTarget();
  const doc = new EventTarget(); doc.hidden = false;
  let attempts = 0;
  const { loader, state } = fixture(null, undefined, {
    window: win, document: doc, retryDelays: [5],
    requestJSON: async () => {
      if (++attempts === 1) throw Object.assign(new Error("sign in"), { status: 401 });
      return { data: [{ id: "books" }] };
    },
  });
  await loader.load();
  win.dispatchEvent(new Event("online"));
  doc.dispatchEvent(new Event("visibilitychange"));
  await new Promise((resolve) => setTimeout(resolve, 30));
  assert.equal(attempts, 1);
  await loader.load();
  assert.equal(attempts, 2);
  assert.equal(state.selected, "books");
  loader.dispose();
});

test("a failed startup retries and recovers even without an online event", async () => {
  let attempts = 0;
  let recovered = 0;
  const { loader, state } = fixture(null, undefined, {
    requestJSON: async () => { if (++attempts === 1) throw new Error("offline"); return { data: [{ id: "books" }] }; },
    retryDelays: [10],
    onReady: () => { recovered++; },
  });
  await loader.load();
  await new Promise((resolve) => setTimeout(resolve, 60));
  assert.equal(attempts, 2);
  assert.equal(state.selected, "books");
  assert.equal(recovered, 1);
  loader.dispose();
});

test("initial load restores a visible saved library and refreshes the view", async () => {
  const { calls, loader, state } = fixture("manga");
  await loader.load();
  assert.equal(state.selected, "manga");
  assert.equal(state.ready, true);
  assert.deepEqual(calls.slice(-3), ["toolbar", "alphabet", "library"]);
});

test("hiding the selected library chooses and persists a replacement", () => {
  const { loader, saved, state } = fixture();
  state.all = [{ id: "books" }, { id: "manga" }];
  state.selected = "manga";
  loader.applyVisible([{ id: "books" }]);
  assert.equal(state.selected, "books");
  assert.equal(saved["mochil-library"], "books");
});

test("reload validates an already initialized selection", async () => {
  const { loader, state } = fixture("missing");
  await loader.load();
  assert.equal(state.selected, "books");
});

test("an empty first run opens setup instead of loading a missing library", async () => {
  const { calls, loader, state } = fixture(null, []);
  await loader.load();
  assert.equal(state.selected, "");
  assert.equal(state.ready, true);
  assert.equal(calls.at(-1), "first-run");
  assert.equal(calls.includes("library"), false);
});
