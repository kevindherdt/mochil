import assert from "node:assert/strict";
import test from "node:test";
import { createMusicDiscoveryLoader } from "../static/music-discovery-loader.js";

const deferred = () => {
  let resolve;
  const promise = new Promise((done) => (resolve = done));
  return { promise, resolve };
};

test("superseded discovery responses cannot replace a newer music destination", async () => {
  const pending = Array.from({ length: 4 }, deferred);
  const requests = [...pending];
  const discoveries = [];
  const loader = createMusicDiscoveryLoader({
    requestJSON: () => requests.shift().promise,
    container: { innerHTML: "" },
    setStatus() {},
    getLibrary: () => "music",
    getFavorites: () => [],
    onDiscovery: (value) => discoveries.push(value),
    onRecent() {},
  });
  const ticket = loader.begin();
  const loading = loader.loadDiscovery(ticket);
  loader.cancel();
  for (const request of pending) request.resolve({ data: [{ id: "stale" }] });
  assert.equal(await loading, false);
  assert.deepEqual(discoveries, []);
});

test("current discovery combines bounded server groups and favorites", async () => {
  const responses = [[{ id: "played" }], [{ id: "recent" }], [{ id: "album" }], [{ id: "artist" }]];
  let discovery;
  const container = { innerHTML: "" };
  const loader = createMusicDiscoveryLoader({
    requestJSON: async () => ({ data: responses.shift() }),
    container,
    setStatus() {},
    getLibrary: () => "my music",
    getFavorites: () => Array.from({ length: 12 }, (_, id) => ({ id })),
    onDiscovery: (value) => (discovery = value),
    onRecent() {},
  });
  assert.equal(await loader.loadDiscovery(loader.begin()), true);
  assert.match(container.innerHTML, /music-loading/);
  assert.equal(discovery.favorites.length, 8);
  assert.equal(discovery.albums[0].id, "album");
  assert.deepEqual(discovery.failures, { played: false, recent: false, albums: false, artists: false });
});

test("discovery keeps healthy sections when one request fails", async () => {
  const responses = [
    Promise.resolve({ data: [{ id: "played" }] }),
    Promise.resolve({ data: [{ id: "recent" }] }),
    Promise.reject(new Error("albums unavailable")),
    Promise.resolve({ data: [{ id: "artist" }] }),
  ];
  let discovery;
  const loader = createMusicDiscoveryLoader({
    requestJSON: () => responses.shift(),
    container: { innerHTML: "" },
    setStatus() {},
    getLibrary: () => "music",
    getFavorites: () => [],
    onDiscovery: value => (discovery = value),
    onRecent() {},
  });

  assert.equal(await loader.loadDiscovery(loader.begin()), true);
  assert.equal(discovery.played[0].id, "played");
  assert.equal(discovery.artists[0].id, "artist");
  assert.deepEqual(discovery.albums, []);
  assert.equal(discovery.failures.albums, true);
});

test("returning to Discovery keeps cached shelves visible while refreshing", async () => {
  const pending = Array.from({ length: 4 }, deferred);
  const requests = [...pending];
  const discoveries = [];
  let phase = 0;
  const initial = [[{ id: "played" }], [{ id: "recent" }], [{ id: "album" }], [{ id: "artist" }]];
  const loader = createMusicDiscoveryLoader({
    requestJSON: async () => phase === 0 ? { data: initial.shift() } : requests.shift().promise,
    container: { innerHTML: "rendered discovery" },
    setStatus() {},
    getLibrary: () => "music",
    getFavorites: () => [{ id: "favorite" }],
    onDiscovery: value => discoveries.push(value),
    onRecent() {},
  });

  assert.equal(await loader.loadDiscovery(loader.begin()), true);
  phase = 1;
  const refreshing = loader.loadDiscovery(loader.begin());
  assert.equal(discoveries.length, 2);
  assert.equal(discoveries[1].albums[0].id, "album");
  for (const request of pending) request.resolve({ data: [] });
  assert.equal(await refreshing, true);
  assert.equal(discoveries.length, 3);
});

test("a failed Discovery refresh retains the last healthy section", async () => {
  const responses = [[{ id: "played" }], [{ id: "recent" }], [{ id: "album" }], [{ id: "artist" }]];
  let phase = 0;
  let discovery;
  const loader = createMusicDiscoveryLoader({
    requestJSON: async () => {
      if (phase === 0) return { data: responses.shift() };
      throw new Error("offline");
    },
    container: { innerHTML: "" },
    setStatus() {},
    getLibrary: () => "music",
    getFavorites: () => [],
    onDiscovery: value => (discovery = value),
    onRecent() {},
  });

  await loader.loadDiscovery(loader.begin());
  phase = 1;
  await loader.loadDiscovery(loader.begin());
  assert.equal(discovery.albums[0].id, "album");
  assert.equal(discovery.failures.albums, true);
});
