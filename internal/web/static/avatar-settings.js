// Settings → Avatar: which Live2D model she wears. Models are folders in the avatar folder on the server; this lists
// them and saves the choice straight away with its own Save button (it does not wait for the page's Save).
import { effectiveModuleState } from "./modules-settings.js?v=19";

const tabHTML = `<button data-settings-tab="avatar" data-hint="The speaking character you can show" aria-pressed="false" hidden><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></svg><span>Avatar</span></button>`;

const panelHTML = `
  <section class="settings-panel" data-settings-panel="avatar" hidden>
    <header><h2>Avatar</h2><p>The character shown beside the Hermes chat. Put one folder per Live2D model in the avatar folder on the server, then choose it here.</p></header>
    <div class="system-card provider-fields">
      <section class="provider-field">
        <label>Model<select id="avatar-model" aria-label="Avatar model"></select></label>
        <label>Voice for this avatar<select id="avatar-voice" aria-label="Voice for this avatar"></select></label>
        <div class="provider-field-footer"><small id="avatar-note" role="status"></small><button type="button" id="avatar-save" class="primary" disabled>Save avatar</button></div>
      </section>
    </div>
  </section>`;

export function createAvatarSettings({ document, $, fetchImpl = (...args) => fetch(...args) }) {
  document.querySelector('[data-settings-tab="hermes"]')?.insertAdjacentHTML("afterend", tabHTML);
  document.querySelector("#settings-view").insertAdjacentHTML("beforeend", panelHTML);
  let saved = {}; // the voice each model speaks with
  const voice = () => $("#avatar-voice");
  const select = () => $("#avatar-model"), note = () => $("#avatar-note"), save = () => $("#avatar-save");
  async function refresh() {
    note().textContent = "Looking for models…";
    const response = await fetchImpl("/api/avatar/models", { credentials: "same-origin" }).catch(() => null);
    if (!response?.ok) { note().textContent = "Could not read the avatar folder."; select().replaceChildren(); return; }
    const { models, selected, core, voices = {} } = await response.json();
    saved = voices;
    const narration = await fetchImpl("/api/narration/status", { credentials: "same-origin" }).then((r) => (r.ok ? r.json() : null)).catch(() => null);
    const choices = [["", "Default voice"], ...((narration?.voices || []).map((voice) => [voice.id, voice.name || voice.id]))];
    voice().replaceChildren(...choices.map(([value, text]) => Object.assign(document.createElement("option"), { value, textContent: text })));
    voice().value = saved[selected] || "";
    select().replaceChildren(...models.map((model) => Object.assign(document.createElement("option"), { value: model.id, textContent: model.id === "." ? model.name : model.id, selected: model.id === selected })));
    save().disabled = true;
    note().textContent = !models.length
      ? "No models found. Add a folder that contains a .model3.json file."
      : core ? `${models.length} model${models.length === 1 ? "" : "s"} found.` : "Models found, but the Live2D Cubism core is missing: put live2dcubismcore.min.js in the avatar folder's _runtime folder.";
  }
  return {
    bind() {
      select().onchange = () => { voice().value = saved[select().value] || ""; save().disabled = false; note().textContent = "Not saved yet."; };
      voice().onchange = () => { save().disabled = false; note().textContent = "Not saved yet."; };
      save().onclick = async () => {
        save().disabled = true;
        note().textContent = "Saving…";
        const response = await fetchImpl("/api/avatar/select", { method: "PUT", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ model: select().value, voice: voice().value }) }).catch(() => null);
        if (response?.ok) saved = (await response.json()).voices || {};
        note().textContent = response?.ok ? "Saved. The chat uses it next time it opens." : "Could not save the avatar.";
        save().disabled = Boolean(response?.ok);
      };
    },
    load(settings, modules = []) {
      const module = modules.find((item) => item.id === "avatar");
      const hermes = modules.find((item) => item.id === "hermes");
      const on = Boolean(module) && effectiveModuleState(module, settings) && Boolean(hermes) && effectiveModuleState(hermes, settings);
      $('[data-settings-tab="avatar"]').hidden = !on;
      if (on) void refresh();
    },
  };
}
