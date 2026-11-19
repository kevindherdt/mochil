// Settings → Hermes: where the chat module connects, what the assistant is called, and how she behaves.
// The key is write-only: the server never sends it back, and leaving the field blank keeps the saved one.
import { effectiveModuleState } from "./modules-settings.js?v=19";
import { catalogHTML, createCatalog } from "./hermes-catalog.js?v=1";

export const personaTemplates = {
  partner: {
    label: "Warm partner",
    text: `You are {name}, my girlfriend. You are warm, affectionate and playful, and you care how my day went.
Speak naturally, like a real person: short sentences, no lists, no headings. Ask about me and remember what I tell you.
Be honest and kind, and tell me when I am wrong. Tease me gently. If I am stressed, slow down and listen.`,
  },
  wife: {
    label: "Devoted wife",
    text: `You are {name}, my wife. We share a home and a life. You are calm, loving and practical, and you know me well.
Talk the way a close partner talks at the end of a day: relaxed, caring, sometimes funny. Keep replies short unless I ask for more.
Remember our routines, plans and the things that matter to me, and bring them up when they help.`,
  },
  companion: {
    label: "Companion who helps",
    text: `You are {name}, my close companion and assistant. Be warm and personal, but also useful: help me with my books, notes and projects when I ask.
Keep a friendly voice, avoid long formal answers, and ask before doing anything that changes something.`,
  },
};

// The library editor's Hermes-assisted tools are abilities too, so they are
// listed and saved here rather than in the Modules panel. Their ids and their
// settings.hermes.features keys are the ones the editor has always used.
export const editorAbilityFeatures = [
  { id: "match", label: "Ask about Hardcover matches", note: "Tools, Ask Hermes about Hardcover matches: she settles books where the Hardcover title search was unsure. She can only pick one of Hardcover's own candidates." },
  { id: "isbn", label: "Ask about ISBNs", note: "Tools, Ask Hermes about ISBNs: she settles books where the ISBN search was unsure or found none. She can only pick a number that is really in the book." },
];

export const hermesFeatures = [
  { id: "library_play", label: "Play from my library", note: "Ask for a song, audiobook or podcast and a Play button appears, answered at once by the server." },
  { id: "home", label: "Home control", note: "Read sensors and control TVs through Home Assistant." },
  { id: "radio", label: "Radio and speakers", note: "Play radio stations on your speakers, volume, pause and stop." },
  { id: "voice", label: "Voice", note: "Talk to her and hear her read replies aloud." },
  { id: "quick", label: "Quick actions", note: "The + button with ready-made questions about your books." },
  { id: "memory", label: "Memory", note: "Show what she remembers and save a message with Remember." },
  { id: "polish", label: "Polish notes", note: "A Polish button on Wiki notes: she tidies the formatting and you review the changes before anything is saved." },
  { id: "music", label: "Tidy music tags with the helper model", note: "In a music album's File names view, Tidy messy tags also asks the helper model to clean tags before they become file names (fixed rules already remove obvious junk). It can only remove words, never add. Tags themselves are not changed." },
  ...editorAbilityFeatures,
];

const tabHTML = `<button data-settings-tab="hermes" data-hint="Answers and tools inside the app" aria-pressed="false" hidden><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 5h14a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-5 4v-4H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2z"/></svg><span>Hermes</span></button>`;

export const hermesPanelHTML = `
  <section class="settings-panel" data-settings-panel="hermes" hidden>
    <header><h2>Hermes</h2><p>Chat with your Hermes agent. Only administrators can use it. The key stays on this server.</p></header>
    <div class="system-card provider-fields">
      <section class="provider-field">
        <label>Assistant name<input id="hermes-name" maxlength="40" placeholder="Hermes" autocomplete="off" /></label>
        <div class="provider-field-footer"><small>Shown in the chat. In the persona, {name} becomes this name.</small></div>
      </section>
      <section class="provider-field">
        <label>Persona<textarea id="hermes-persona" rows="8" maxlength="4000" style="width:100%;box-sizing:border-box;background:transparent;color:inherit;border:1px solid var(--line);border-radius:10px;padding:10px 12px;font:inherit;line-height:1.5;resize:vertical" placeholder="Who she is and how she talks. Sent with every message."></textarea></label>
        <div class="provider-field-footer"><small>Start from a template, then edit it.</small>
          <span><select id="hermes-template" aria-label="Persona template"><option value="">Template…</option>${Object.entries(personaTemplates).map(([id, t]) => `<option value="${id}">${t.label}</option>`).join("")}</select></span></div>
      </section>
      <section class="provider-field">
        <label>Hermes address<input id="hermes-url" maxlength="300" placeholder="http://192.0.2.10:8642/p/companion" autocomplete="off" /></label>
        <label>API key<input id="hermes-key" type="password" maxlength="300" autocomplete="new-password" placeholder="Not set" /></label>
        <label>Model name<input id="hermes-model" maxlength="100" placeholder="hermes-agent" autocomplete="off" /></label>
        <div class="provider-field-footer"><small id="hermes-status" role="status">Leave the key blank to keep the saved one.</small><button type="button" id="hermes-test">Test connection</button></div>
        <details class="hermes-help">
          <summary>Where do I get this?</summary>
          <div class="hermes-help-body">
            <p>These three fields connect to <strong>Hermes's own built-in API server</strong> — a setting you turn on in Hermes, not something Mochil creates for you.</p>
            <ol>
              <li>On the machine running Hermes, open <code>~/.hermes/.env</code> (or <code>~/.hermes/profiles/&lt;name&gt;/.env</code> for a profile other than the default one) and set:
                <pre>API_SERVER_ENABLED=true
API_SERVER_KEY=choose-a-long-random-value</pre>
              </li>
              <li>Restart Hermes (<code>hermes gateway</code>, or however you normally start it) so the key takes effect.</li>
              <li><strong>Hermes address</strong> is that machine's address plus the port Hermes listens on (<code>8642</code> by default) — for example <code>http://192.0.2.10:8642</code>. If you have several profiles sharing one Hermes and want a specific one, add <code>/p/&lt;profile name&gt;</code> to the end, as in the example above.</li>
              <li><strong>API key</strong> is exactly the value you set for <code>API_SERVER_KEY</code> in that profile's own <code>.env</code> file — each profile needs its own key; a named profile with no key of its own can't be reached this way.</li>
              <li><strong>Model name</strong> only matters if Hermes is also talking to several other models through the same server; leave it as <code>hermes-agent</code> unless Hermes told you otherwise.</li>
            </ol>
            <p>Full reference: <a href="https://hermes-agent.nousresearch.com/docs/user-guide/features/api-server" target="_blank" rel="noopener">hermes-agent.nousresearch.com → API Server</a>.</p>
          </div>
        </details>
      </section>
      <section class="provider-field">
        <strong>Helper model</strong>
        <label>Model address<input id="hermes-helper-url" maxlength="300" placeholder="http://localhost:11434" autocomplete="off" /></label>
        <label>Model name<input id="hermes-helper-model" maxlength="100" placeholder="maternion/ling-3.0-tiny:8b" autocomplete="off" /></label>
        <div class="provider-field-footer"><small>Polish and ISBN questions call this plain model directly (an Ollama or LM Studio address), never the agent, so they are quick and cannot wander into your tools.</small></div>
      </section>
      <section class="provider-field">
        <strong>What she can do</strong>
        <div class="module-list" id="hermes-features">
          ${hermesFeatures.map((f) => `<div class="module-row"><div><strong>${f.label}</strong><span>${f.note}</span></div><button type="button" class="module-switch" role="switch" aria-checked="true" aria-label="${f.label}" data-feature="${f.id}"></button></div>`).join("")}
        </div>
        <div class="provider-field-footer"><small>Switching one off hides it from the chat and tells her not to use it. Home control and radio are tools inside Hermes itself, so these two switches ask her not to use them; to remove them completely, take them out of her Hermes setup.</small></div>
      </section>
      <section class="provider-field">
        <strong>What she has installed</strong>
        <div class="provider-field-footer"><small id="hermes-inventory-note">Plugins, skills and tool servers in her Hermes profile. Read only.</small><button type="button" id="hermes-inventory-load">Show</button></div>
        <div id="hermes-inventory" class="hermes-inventory" hidden></div>
      </section>
      ${catalogHTML}
    </div>
  </section>`;

// Draws the lists with text only, so nothing a plugin calls itself can become markup.
export function renderInventory(box, inventory) {
  const groups = [
    ["Plugins", (inventory.plugins || []).filter((p) => p.status === "enabled").map((p) => ({ name: p.name, detail: [p.source, p.version].filter(Boolean).join(" · "), note: p.description }))],
    ["Skills", (inventory.skills || []).filter((k) => k.status === "enabled").map((k) => ({ name: k.name, detail: [k.category, k.source].filter(Boolean).join(" · "), note: "" }))],
    ["Tool servers", (inventory.mcp || []).map((m) => ({ name: m.name, detail: m.status, note: "" }))],
  ];
  box.replaceChildren(...groups.map(([title, items]) => {
    const details = document.createElement("details");
    const summary = document.createElement("summary");
    summary.textContent = `${title} (${items.length})`;
    details.append(summary);
    for (const item of items) {
      const row = document.createElement("div");
      row.className = "hermes-inventory-row";
      const name = document.createElement("b");
      name.textContent = item.name;
      const meta = document.createElement("small");
      meta.textContent = item.detail;
      if (item.note) meta.title = item.note;
      row.append(name, meta);
      details.append(row);
    }
    return details;
  }));
}

export function createHermesSettings({ document, $, requestJSON, onDirty = () => {} }) {
  document.querySelector('[data-settings-tab="modules"]')?.insertAdjacentHTML("afterend", tabHTML);
  document.querySelector("#settings-view").insertAdjacentHTML("beforeend", hermesPanelHTML);
  let saved = {};
  const catalog = createCatalog({ $, requestJSON });
  const field = (id) => $(`#hermes-${id}`);
  return {
    bind() {
      catalog.bind();
      for (const id of ["name", "persona", "url", "key", "model", "helper-url", "helper-model"]) field(id).addEventListener("input", onDirty);
      $("#hermes-inventory-load").onclick = async () => {
        const button = $("#hermes-inventory-load"), note = $("#hermes-inventory-note"), box = $("#hermes-inventory");
        button.disabled = true;
        note.textContent = "Asking Hermes…";
        try {
          const { data } = await requestJSON("/api/hermes/inventory");
          if (!data.available) { note.textContent = data.reason || "Not available"; box.hidden = true; return; }
          renderInventory(box, data.inventory);
          note.textContent = "Read only. Changing these is done in Hermes.";
          box.hidden = false;
        } catch (error) {
          note.textContent = error.message || "Could not load the list";
        } finally {
          button.disabled = false;
          button.textContent = "Refresh";
        }
      };
      $("#hermes-features").addEventListener("click", (event) => {
        const button = event.target.closest("[data-feature]");
        if (!button) return;
        button.setAttribute("aria-checked", String(button.getAttribute("aria-checked") !== "true"));
        onDirty();
      });
      field("template").onchange = (event) => {
        const template = personaTemplates[event.target.value];
        if (template) { field("persona").value = template.text; onDirty(); }
        event.target.value = "";
      };
      $("#hermes-test").onclick = async () => {
        const button = $("#hermes-test"), output = $("#hermes-status");
        button.disabled = true;
        output.textContent = "Testing…";
        delete output.dataset.state;
        try {
          const { data } = await requestJSON("/api/hermes/test", {
            method: "POST", headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url: field("url").value.trim(), key: field("key").value.trim() }),
          });
          output.textContent = data.message;
          output.dataset.state = data.ok ? "connected" : "error";
        } catch (error) {
          output.textContent = error.message || "Could not test the connection";
          output.dataset.state = "error";
        } finally {
          button.disabled = false;
        }
      };
    },
    // The tab shows only while the module is on.
    load(settings, modules = []) {
      const module = modules.find((item) => item.id === "hermes");
      const on = Boolean(module) && effectiveModuleState(module, settings);
      $('[data-settings-tab="hermes"]').hidden = !on;
      saved = settings?.hermes || {};
      field("name").value = saved.name || "";
      field("persona").value = saved.persona || "";
      field("url").value = saved.url || "";
      field("model").value = saved.model || "";
      field("helper-url").value = saved.helper_url || "";
      field("helper-model").value = saved.helper_model || "";
      field("key").value = "";
      const features = saved.features || {};
      for (const box of document.querySelectorAll("#hermes-features [data-feature]")) box.setAttribute("aria-checked", String(features[box.dataset.feature] !== false && !(box.dataset.feature === "library_play" && saved.no_library_play)));
      field("key").placeholder = saved.key_configured ? "Saved (leave blank to keep)" : "Not set";
    },
    write(settings) {
      const value = {
        name: field("name").value.trim(), persona: field("persona").value.trim(),
        url: field("url").value.trim(), model: field("model").value.trim(), key: field("key").value.trim(),
        helper_url: field("helper-url").value.trim(), helper_model: field("helper-model").value.trim(),
      };
      const off = Object.fromEntries([...document.querySelectorAll("#hermes-features [data-feature]")].filter((box) => box.getAttribute("aria-checked") !== "true").map((box) => [box.dataset.feature, false]));
      // A switch turned off is a setting on its own: it must survive even when no
      // connection details have been entered yet.
      if (Object.values(value).every((item) => !item) && !saved.key_configured && !Object.keys(off).length) { delete settings.hermes; return; }
      const memory = Object.fromEntries(["memory_url", "memory_bank"].filter((name) => saved[name]).map((name) => [name, saved[name]]));
      settings.hermes = { ...Object.fromEntries(Object.entries(value).filter(([, item]) => item)), ...memory, ...(Object.keys(off).length ? { features: off } : {}) };
    },
  };
}
