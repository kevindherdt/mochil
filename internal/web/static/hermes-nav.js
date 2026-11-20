// The chat's way back into the rest of Mochil: the same Home / library menu the other screens have, at the top left.
// It reuses the shared menu component, so it looks and behaves the same; choosing something leaves the chat.
import { createLibraryMenu } from "./library-menu.js?v=7";

const DESTINATIONS = { __home__: "/", __settings__: "/#/settings" };

async function libraries(fetchImpl) {
  try {
    const response = await fetchImpl("/api/libraries");
    const list = response.ok ? await response.json() : [];
    return Array.isArray(list) ? list.filter((library) => library && library.id && library.name) : [];
  } catch (_) {
    return [];
  }
}

export async function mountHermesNav({ document = globalThis.document, fetchImpl = (...args) => fetch(...args), navigate = (url) => globalThis.location.assign(url) } = {}) {
  const bar = document.querySelector(".hm-bar");
  if (!bar || bar.querySelector(".hm-nav")) return null;
  const host = document.createElement("label");
  host.className = "shelf-switcher hm-nav";
  const caption = document.createElement("span");
  caption.className = "sr-only";
  caption.textContent = "Home or library";
  const select = document.createElement("select");
  select.setAttribute("aria-label", "Choose Home or a library");
  host.append(caption, select);
  bar.prepend(host);
  const option = (value, text, kind, count) => {
    const item = document.createElement("option");
    item.value = value;
    item.textContent = text;
    item.dataset.kind = kind;
    if (count != null) item.dataset.count = String(count);
    return item;
  };
  const all = await libraries(fetchImpl);
  select.append(option("__home__", "Home", "home"), ...all.map((library) => option(library.id, library.name, library.kind || "library", library.count)), option("__hermes__", "Hermes chat", "hermes"), option("__settings__", "Settings", "settings"));
  select.value = "__hermes__";
  createLibraryMenu({ document, select });
  select.addEventListener("change", () => {
    const value = select.value;
    if (value === "__hermes__") return;
    navigate(DESTINATIONS[value] || `/#/library/${encodeURIComponent(value)}`);
  });
  return host;
}

if (typeof document !== "undefined" && document.querySelector?.(".hm-bar")) void mountHermesNav();
