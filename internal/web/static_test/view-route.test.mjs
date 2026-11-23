import assert from "node:assert/strict";
import test from "node:test";
import { createViewRoute, parsePlace } from "../static/view-route.js";

function fixture({ hash = "", libraries = ["books", "wiki"] } = {}) {
  const state = { view: "dashboard", library: "books", calls: [], pushed: [], places: [], place: {} };
  const listeners = {};
  const window = {
    location: { hash, pathname: "/", search: "" },
    history: {
      pushState: (_s, _t, url) => { state.pushed.push(url); window.location.hash = url.slice(url.indexOf("#")); },
      replaceState: (_s, _t, url) => { state.replaced = url; window.location.hash = url; },
    },
    addEventListener: (type, fn) => (listeners[type] = fn),
  };
  const route = createViewRoute({
    window,
    getView: () => state.view,
    getLibrary: () => state.library,
    hasLibrary: (id) => libraries.includes(id),
    showHome: () => { state.view = "dashboard"; state.calls.push("home"); },
    showSettings: () => { state.view = "settings"; state.calls.push("settings"); },
    showLibrary: (id, place) => { state.view = "library"; state.library = id; state.places.push(place); state.calls.push(`library:${id}`); },
    getPlace: () => state.place,
  });
  return { state, route, window, listeners };
}

test("deferred initial restoration never overrides a user's new route", () => {
  const f = fixture({ hash: "#/library/wiki/collections/abc" });
  f.state.view = "settings";
  f.route.write();
  f.route.restoreInitial();
  assert.deepEqual(f.state.calls, []);
  assert.equal(f.window.location.hash, "#/settings");
});

test("a library hash navigated to before discovery is restored when ready", () => {
  const libraries = [];
  const f = fixture({ libraries });
  f.window.location.hash = "#/library/wiki/collections/abc";
  f.listeners.hashchange();
  libraries.push("wiki");
  assert.equal(f.route.restoreInitial(), true);
  assert.equal(f.state.view, "library");
  assert.equal(f.state.library, "wiki");
  assert.equal(f.state.places.at(-1).collection, "abc");
});

test("deferred initial restoration restores the original library place", () => {
  const f = fixture({ hash: "#/library/wiki/collections/abc" });
  assert.equal(f.route.restoreInitial(), true);
  assert.deepEqual(f.state.calls, ["library:wiki"]);
  assert.equal(f.state.places[0].collection, "abc");
});

test("views are written to the hash once and not duplicated", () => {
  const { state, route, window } = fixture();
  state.view = "settings";
  route.write();
  route.write();
  assert.deepEqual(state.pushed, ["#/settings"]);
  state.view = "library";
  state.library = "wiki";
  route.write();
  assert.equal(window.location.hash, "#/library/wiki");
  state.view = "dashboard";
  route.write();
  assert.equal(window.location.hash, "#/");
});

test("refresh restores settings and libraries from the hash", () => {
  let f = fixture({ hash: "#/settings" });
  f.route.restore();
  assert.deepEqual(f.state.calls, ["settings"]);
  f = fixture({ hash: "#/library/wiki" });
  f.route.restore();
  assert.deepEqual(f.state.calls, ["library:wiki"]);
  assert.deepEqual(f.state.pushed, [], "restoring never adds history entries");
});

test("unknown libraries fall back to Home", () => {
  const f = fixture({ hash: "#/library/gone" });
  f.route.restore();
  assert.deepEqual(f.state.calls, ["home"]);
});

test("an empty hash opens the library, like the root", () => {
  const f = fixture();
  f.route.restore();
  assert.deepEqual(f.state.calls, ["library:books"]);
});

test("Back and Forward re-apply the hash", () => {
  const f = fixture({ hash: "#/library/books" });
  f.listeners.popstate();
  assert.deepEqual(f.state.calls, ["library:books"]);
  // The root address is the library you were last in: the dock has no Home, so #/ must not dump you on a dashboard.
  f.window.location.hash = "#/";
  f.listeners.hashchange();
  assert.deepEqual(f.state.calls, ["library:books", "library:books"]);
});

test("the root address opens a library, not the dashboard", () => {
  const f = fixture({ hash: "#/" });
  f.route.restore();
  assert.deepEqual(f.state.calls, ["library:books"]);
  assert.equal(f.state.view, "library");
});

test("with no usable shelf the root falls back to the dashboard rather than a blank page", () => {
  const f = fixture({ hash: "#/", libraries: [] });
  f.state.library = "gone";
  f.state.view = "library";
  f.route.restore();
  assert.deepEqual(f.state.calls, ["home"]);
});

test("the tab and an opened collection are kept in the hash and restored after a refresh", () => {
  const { state, route, window } = fixture();
  state.view = "library";
  state.library = "wiki";
  state.place = { tab: "collections", collection: "abc 1" };
  route.sync();
  assert.equal(window.location.hash, "#/library/wiki/collections/abc%201");
  assert.deepEqual(state.pushed, [], "changing the tab never adds a history entry");
  const again = fixture({ hash: "#/library/wiki/collections/abc%201" });
  again.route.restore();
  assert.deepEqual(again.state.places, [{ tab: "collections", collection: "abc 1", detail: "" }]);
  assert.deepEqual(parsePlace("authors"), { tab: "authors", collection: "", detail: "" });
  assert.deepEqual(parsePlace("nonsense/x"), { tab: "", collection: "", detail: "" });
});

test("an opened series or author is named in the address and read back from it", () => {
  assert.deepEqual(parsePlace("series/The%20Expanse"), { tab: "series", collection: "", detail: "The Expanse" });
  assert.deepEqual(parsePlace("authors/Frank%20Herbert"), { tab: "authors", collection: "", detail: "Frank Herbert" });
});
