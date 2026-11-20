// Settings → Modules: switch optional parts of Mochil on or off. The core
// (books, comics, manga, reading position, sync, accounts) is always on.
import { iconForModule } from "./settings-icons.js?v=1";
const escapeHTML = (value) =>
  String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]);

export const modulesPanelHTML = `
  <section class="settings-panel" data-settings-panel="modules" hidden>
    <header><h2>Modules</h2><p>Turn optional features on or off. A module that is off does not run at all.</p></header>
    <div class="system-card module-list" id="module-list"></div>
    <p class="module-note">Books, comics and manga, reading sync and accounts are always on. Turning a module off hides its libraries and stops its background work. Nothing is deleted.</p>
  </section>`;

export function mountModulesSettings(document) {
  if (document.querySelector('[data-settings-panel="modules"]')) return;
  document.querySelector("#settings-view").insertAdjacentHTML("beforeend", modulesPanelHTML);
  // The library editor's options live with the module that switches it on; its section is static markup in index.html
  // because editor-link.js wires the menu checkbox before this lazy module exists, and moving a node keeps that wiring.
  const editor = document.querySelector('[data-settings-panel="editor"]');
  if (editor) document.querySelector('[data-settings-panel="modules"]')?.append(editor);
}

// A module's on/off state is decided authoritatively by the server (GET /api/modules'
// `enabled` field, via moduleOn on the backend) so the toggle can never show a state
// the server itself would not honour. The settings-based heuristic only remains as a
// fallback for callers (and tests) that pass a bare module list without that field.
export function effectiveModuleState(module, settings) {
  if (typeof module?.enabled === "boolean") return module.enabled;
  const explicit = settings?.modules?.[module.id];
  if (typeof explicit === "boolean") return explicit;
  if (module.default_on) return true;
  return (settings?.libraries || []).some((library) => (module.kinds || []).includes(library.kind));
}

export function moduleLibraryCount(module, settings) {
  return (settings?.libraries || []).filter((library) => (module.kinds || []).includes(library.kind)).length;
}

function moduleUsage(module, enabled, count) {
  if (!(module.kinds || []).length) return "";
  return count === 0 ? "No libraries use it yet" : `${count} ${count === 1 ? "library" : "libraries"}${enabled ? "" : " hidden while off"}`;
}

function moduleSwitchHTML(module, enabled, label = module.name) {
  return `<button type="button" class="module-switch" role="switch" aria-checked="${enabled}" aria-label="${escapeHTML(label)}" data-module="${escapeHTML(module.id)}"></button>`;
}

export function moduleRowHTML(module, enabled, count) {
  const usage = moduleUsage(module, enabled, count);
  return `<div class="module-row"><i class="ico" data-icon="${iconForModule(module.id)}" aria-hidden="true"></i><div><strong>${escapeHTML(module.name)}<em class="module-pill" data-state="${enabled ? "on" : "off"}">${enabled ? "Running" : "Off"}</em></strong><span>${escapeHTML(module.description)}</span>${usage ? `<small>${usage}</small>` : ""}</div>${moduleSwitchHTML(module, enabled)}</div>`;
}

// A module can be one card with options inside it. The server marks those options
// with {group, option: true}, so the card keeps its own switch and the options
// sit under it instead of reading as separate modules.
export function groupedModules(modules = []) {
  const childrenByGroup = new Map();
  for (const module of modules)
    if (module.option && module.group && module.group !== module.id) {
      const children = childrenByGroup.get(module.group) || [];
      children.push(module);
      childrenByGroup.set(module.group, children);
    }
  const childIds = new Set([...childrenByGroup.values()].flat().map((module) => module.id));
  const groups = modules
    .filter((module) => !childIds.has(module.id))
    .map((module) => ({ module, children: childrenByGroup.get(module.id) || [] }));
  // An option whose parent was not sent must not vanish from Settings; it falls
  // back to a row of its own.
  for (const children of childrenByGroup.values())
    for (const module of children)
      if (!modules.some((other) => other.id === module.group))
        groups.push({ module, children: [] });
  return groups;
}

// "Hardcover shelf sync" reads as three modules; the card is simply Hardcover and
// its description is what shelf sync does.
const groupTitles = { hardcover: "Hardcover" };

function groupDescription(parent) {
  if (parent.id !== "hardcover") return parent.description;
  const description = String(parent.description || "").trim();
  if (/shelf/i.test(description)) return description;
  return `Shelf sync: ${description || "mark books as currently reading or read on your Hardcover account."}`;
}

export function moduleGroupHTML(parent, children, state, count) {
  const enabled = Boolean(state[parent.id]);
  const usage = moduleUsage(parent, enabled, count);
  const title = groupTitles[parent.id] || parent.name;
  return `<div class="module-row module-group"><div class="module-group-head"><i class="ico" data-icon="${iconForModule(parent.id)}" aria-hidden="true"></i><div><strong>${escapeHTML(title)}<em class="module-pill" data-state="${enabled ? "on" : "off"}">${enabled ? "Running" : "Off"}</em></strong><span>${escapeHTML(groupDescription(parent))}</span>${usage ? `<small>${usage}</small>` : ""}</div>${moduleSwitchHTML(parent, enabled, title)}</div><div class="module-group-children">${children.map((child) => moduleOptionHTML(child, state[child.id], !enabled)).join("")}</div></div>`;
}

// Turning the card off only disables its options; their saved state stays in the
// settings document, so turning it back on restores exactly what was chosen.
// While it is off the switch must not announce "on": the option does nothing, so
// aria-checked follows what actually runs and the saved choice is named in the
// label instead.
export function moduleOptionHTML(module, enabled, disabled = false) {
  const label = disabled && enabled
    ? `${module.name}, saved but off until its module is on`
    : module.name;
  return `<div class="module-row module-sub"${disabled ? ' data-disabled="true"' : ""}><div><strong>${escapeHTML(module.name)}</strong><span>${escapeHTML(module.description)}</span></div><button type="button" class="module-switch" role="switch" aria-checked="${enabled && !disabled}" aria-label="${escapeHTML(label)}" data-module="${escapeHTML(module.id)}" data-parent="${escapeHTML(module.group || "")}"${disabled ? " disabled" : ""}></button></div>`;
}

export function moduleRowsHTML(modules = [], state = {}, settings = {}) {
  return groupedModules(modules)
    .map(({ module, children }) =>
      children.length
        ? moduleGroupHTML(module, children, state, moduleLibraryCount(module, settings))
        : moduleRowHTML(module, state[module.id], moduleLibraryCount(module, settings)),
    )
    .join("");
}

export function createModulesSettings({ document, $, onDirty = () => {} }) {
  mountModulesSettings(document);
  let modules = [];
  let settings = {};
  let state = {};
  const touched = new Set();

  function render() {
    $("#module-list").innerHTML = moduleRowsHTML(modules, state, settings);
  }

  return {
    bind() {
      $("#module-list").addEventListener("click", (event) => {
        const button = event.target.closest("[data-module]");
        // A disabled option is only waiting for its card to come back on.
        if (!button || button.disabled) return;
        state[button.dataset.module] = !state[button.dataset.module];
        touched.add(button.dataset.module);
        // Their tabs follow the switch at once, without waiting for Save. Avatar needs Hermes.
        for (const id of ["hermes", "avatar"]) {
          const tab = $(`[data-settings-tab="${id}"]`);
          if (tab) tab.hidden = !(state[id] && (id !== "avatar" || state.hermes));
        }
        const editorSection = $('[data-settings-panel="editor"]');
        if (editorSection) editorSection.hidden = !state.editor;
        render();
        onDirty();
        $(`[data-module="${button.dataset.module}"]`)?.focus();
      });
    },
    load(nextModules, nextSettings) {
      modules = nextModules || [];
      settings = nextSettings || {};
      touched.clear();
      state = Object.fromEntries(modules.map((module) => [module.id, effectiveModuleState(module, settings)]));
      render();
    },
    // True when a switch has been moved since the settings were loaded.
    changed() { return touched.size > 0; },
    // Called once the settings have been saved. The moved switches have been written, and the restart
    // prompt has been offered for them; keeping the set meant every later save in the same visit
    // claimed a module had just changed and asked for a restart again, over a change that was already
    // applied.
    saved() {
      touched.clear();
      // The switches just written are now the saved values, so a later comparison starts from them
      // rather than from what was loaded when the page opened.
      settings = { ...(settings || {}), modules: { ...(settings?.modules || {}) } };
      for (const module of modules) settings.modules[module.id] = state[module.id];
    },
    // Only switches the user actually moved become explicit settings; the rest
    // keep following their libraries.
    write(target) {
      if (!touched.size) return;
      target.modules = { ...(target.modules || {}) };
      for (const id of touched) target.modules[id] = state[id];
    },
  };
}
