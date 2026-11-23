// Keeps the current view in the URL hash so refresh, Back and Forward land on
// the same place: "#/" Home, "#/settings", "#/library/<id>", and inside a
// library the tab and an opened collection: "#/library/<id>/collections/<collection>".
const TABS = ["discover", "series", "authors", "books", "collections"];

export function parsePlace(rest) {
  const [tab, collection] = rest.split("/").map((part) => decodeURIComponent(part || ""));
  return { tab: TABS.includes(tab) ? tab : "", collection: tab === "collections" ? collection : "", detail: tab === "series" || tab === "authors" ? collection || "" : "" };
}

export function createViewRoute({ window, getView, getLibrary, hasLibrary, showHome, showSettings, showLibrary, getPlace = () => ({}) }) {
  let applying = false;
  const initialHash = window.location.hash;
  let initialPending = true;
  let pendingLibraryHash;

  function current() {
    const view = getView();
    if (view === "settings") return "#/settings";
    if (view === "library" && getLibrary()) {
      const { tab, collection, detail } = getPlace();
      const named = collection || detail;
      const place = tab ? `/${tab}${named ? `/${encodeURIComponent(named)}` : ""}` : "";
      return `#/library/${encodeURIComponent(getLibrary())}${place}`;
    }
    return "#/";
  }

  function write() {
    if (applying) return;
    const next = current();
    const hash = window.location.hash || "#/";
    if (next === hash) return;
    // Home is the root: only keep a hash while somewhere else.
    window.history.pushState(null, "", next === "#/" ? window.location.pathname + window.location.search + "#/" : next);
  }

  function restore() {
    const hash = window.location.hash || "#/";
    pendingLibraryHash = undefined;
    applying = true;
    try {
      if (hash === "#/settings") showSettings();
      else if (hash.startsWith("#/library/")) {
        const [rawId, ...rest] = hash.slice("#/library/".length).split("/");
        const id = decodeURIComponent(rawId);
        if (hasLibrary(id)) { showLibrary(id, parsePlace(rest.join("/"))); return true; }
        else { if (initialPending) pendingLibraryHash = hash; showHome(); }
      } else if (!hash || hash === "#/" || hash === "#") {
        // Library is the landing page now: the dock has no Home, so the root address opens the shelf you were last
        // in rather than a dashboard nothing points at.
        const last = getLibrary();
        if (last && hasLibrary(last)) showLibrary(last, parsePlace(""));
        else if (getView() !== "dashboard") showHome();
      } else if (getView() !== "dashboard") showHome();
    } finally {
      applying = false;
    }
  }

  function restoreInitial() {
    if (!initialPending) return;
    initialPending = false;
    if (window.location.hash === initialHash || window.location.hash === pendingLibraryHash) return restore();
  }

  // The tab or an opened collection changed: update the address without adding a history entry.
  function sync() {
    if (applying || getView() !== "library") return;
    const next = current();
    if (next !== (window.location.hash || "#/")) window.history.replaceState(null, "", next);
  }

  window.addEventListener("mochil:place", sync);
  window.addEventListener("popstate", restore);
  window.addEventListener("hashchange", restore);
  return { write, restore, restoreInitial, sync };
}
