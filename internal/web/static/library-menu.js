// Replaces the browser's native <select> popup with a small themed listbox.
// The select stays in the page as the single source of truth (it is hidden from
// view and assistive tech), so every existing handler keeps working unchanged.
import { kindIconSVG } from "./kind-icons.js?v=2";
import { isLibraryDestination, libraryHue } from "./library-identity.js?v=2";

export function createLibraryMenu({ document, select }) {
  // The wrapper is what the layout styles, but a page rendered by an older shell can hand us a bare
  // select: falling back to its parent keeps the enhancement working instead of silently leaving the
  // browser's own grey popup in place, which is what a stale shell looked like.
  const host = select.closest(".shelf-switcher") || select.parentElement;
  if (!host || host.dataset.menu) return null;
  host.dataset.menu = "ready";
  host.classList.add("library-menu-enhanced");
  select.tabIndex = -1;
  select.setAttribute("aria-hidden", "true");
  // Hidden here as well as in library-menu.css: the stylesheet rule needs the class above, so a page
  // that loads an older stylesheet would otherwise show the native select on top of the themed one.
  select.style.position = "absolute";
  select.style.inset = "0";
  select.style.opacity = "0";
  select.style.pointerEvents = "none";

  const button = document.createElement("button");
  button.type = "button";
  button.className = "library-menu-button";
  button.setAttribute("aria-haspopup", "listbox");
  button.setAttribute("aria-expanded", "false");
  const mark = document.createElement("i");
  mark.className = "library-menu-mark";
  const label = document.createElement("span");
  button.append(mark, label);
  const list = document.createElement("ul");
  list.className = "library-menu-list";
  list.setAttribute("role", "listbox");
  list.setAttribute("aria-label", "Home or library");
  list.hidden = true;
  host.append(button, list);

  let items = [];
  let active = -1;

  function sync() {
    items = [...select.options].map((option) => {
      const item = document.createElement("li");
      item.setAttribute("role", "option");
      item.dataset.value = option.value;
      item.id = `library-option-${option.value}`;
      const name = document.createElement("span");
      name.textContent = option.textContent;
      if (option.dataset.kind) {
        item.dataset.kind = option.dataset.kind;
        item.insertAdjacentHTML("afterbegin", kindIconSVG(option.dataset.kind));
      }
      // A library wears its own hue, so two libraries of the same kind are not the same colour.
      // Home, Settings and the editor are destinations rather than libraries and stay neutral.
      if (isLibraryDestination(option.value)) {
        item.dataset.library = option.value;
        item.style.setProperty("--kind-h", String(libraryHue(option.value)));
      }
      item.append(name);
      if (option.dataset.count) {
        const count = document.createElement("small");
        count.textContent = Number(option.dataset.count).toLocaleString();
        item.append(count);
      }
      item.setAttribute("aria-selected", String(option.value === select.value));
      return item;
    });
    list.replaceChildren(...items);
    label.textContent = select.selectedOptions[0]?.textContent || "";
    const current = select.selectedOptions[0];
    mark.innerHTML = current?.dataset.kind ? kindIconSVG(current.dataset.kind) : "";
    mark.dataset.kind = current?.dataset.kind || "";
    // The button shows the same library, so it shows the same hue.
    if (current && isLibraryDestination(current.value)) {
      mark.dataset.library = current.value;
      mark.style.setProperty("--kind-h", String(libraryHue(current.value)));
    } else {
      delete mark.dataset.library;
      mark.style.removeProperty("--kind-h");
    }
    button.setAttribute("aria-label", `${label.textContent}, change library`);
  }

  function highlight(index) {
    active = Math.max(0, Math.min(items.length - 1, index));
    items.forEach((item, i) => item.classList.toggle("active", i === active));
    items[active]?.scrollIntoView({ block: "nearest" });
    button.setAttribute("aria-activedescendant", items[active]?.id || "");
  }

  function open() {
    sync();
    list.hidden = false;
    button.setAttribute("aria-expanded", "true");
    highlight(Math.max(0, [...select.options].findIndex((o) => o.value === select.value)));
  }

  function close(returnFocus = true, ring = true) {
    list.hidden = true;
    button.setAttribute("aria-expanded", "false");
    button.removeAttribute("aria-activedescendant");
    if (returnFocus) button.focus({ focusVisible: ring });
  }

  function choose(index, ring = true) {
    const option = select.options[index];
    close(true, ring);
    if (!option) return;
    select.value = option.value;
    select.dispatchEvent(new Event("change", { bubbles: true }));
  }

  button.onclick = () => (list.hidden ? open() : close());
  list.onclick = (event) => {
    const item = event.target.closest("li");
    if (item) choose(items.indexOf(item), false);
  };
  list.onpointermove = (event) => {
    const item = event.target.closest("li");
    if (item) highlight(items.indexOf(item));
  };
  button.onkeydown = (event) => {
    const keys = { ArrowDown: 1, ArrowUp: -1 };
    if (list.hidden) {
      if (event.key in keys || event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        open();
      }
      return;
    }
    if (event.key in keys) highlight(active + keys[event.key]);
    else if (event.key === "Home") highlight(0);
    else if (event.key === "End") highlight(items.length - 1);
    else if (event.key === "Enter" || event.key === " ") choose(active);
    else if (event.key === "Escape") close();
    else if (event.key === "Tab") close(false);
    else return;
    event.preventDefault();
  };
  document.addEventListener("pointerdown", (event) => {
    if (!list.hidden && !host.contains(event.target)) close(false);
  });
  // Code that focuses the select after a change lands on the visible control.
  select.addEventListener("focus", () => button.focus({ focusVisible: false }));
  new MutationObserver(sync).observe(select, { childList: true });
  select.addEventListener("change", sync);
  sync();
  return { sync, open, close };
}
