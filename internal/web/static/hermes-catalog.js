// Settings → Hermes → "Find more": search the reviewed plugin and skill catalogs and install one, left switched off.
// Only official and verified entries get an Install button; community entries are listed so you can see them, and are
// installed in Hermes itself if you decide to. The helper checks this again before it installs anything.

export const catalogHTML = `
  <section class="provider-field">
    <strong>Find more</strong>
    <div class="hermes-catalog-search">
      <select id="hermes-catalog-kind" aria-label="What to look for"><option value="plugins">Plugins</option><option value="skills">Skills</option></select>
      <input id="hermes-catalog-q" maxlength="80" placeholder="Search, for example home or notes" autocomplete="off" />
      <button type="button" id="hermes-catalog-go">Search</button>
    </div>
    <div class="provider-field-footer"><small id="hermes-catalog-note">Official and verified ones can be installed here. They are added switched off; you turn them on in Hermes and restart it.</small></div>
    <div id="hermes-catalog" class="hermes-inventory" hidden></div>
  </section>`;

const text = (tag, value, className) => { const el = document.createElement(tag); el.textContent = value; if (className) el.className = className; return el; };

export function createCatalog({ $, requestJSON }) {
  const note = (message) => { $("#hermes-catalog-note").textContent = message; };

  function row(kind, item, box) {
    const line = document.createElement("div");
    line.className = "hermes-inventory-row hermes-catalog-row";
    const main = document.createElement("div");
    const tier = item.tier || item.source || "";
    main.append(text("b", item.name), text("small", [tier, item.category, item.capabilities].filter(Boolean).join(" · ")));
    if (item.description) main.append(text("small", item.description, "hermes-catalog-desc"));
    const action = document.createElement("div");
    if (item.installed) action.append(text("small", "Installed"));
    else if (!item.installable) action.append(text("small", "Not reviewed: install it in Hermes"));
    else {
      const button = text("button", "Install");
      button.type = "button";
      button.onclick = () => confirm(kind, item, action);
      action.append(button);
    }
    line.append(main, action);
    return line;
  }

  // Installing is a deliberate two-step: the first tap asks, the second installs.
  function confirm(kind, item, action) {
    const ask = text("small", `Add ${item.name}, switched off?`);
    const yes = text("button", "Install"), no = text("button", "Cancel");
    yes.type = no.type = "button";
    no.onclick = () => action.replaceChildren(...[Object.assign(text("button", "Install"), { type: "button", onclick: () => confirm(kind, item, action) })]);
    yes.onclick = async () => {
      action.replaceChildren(text("small", "Installing…"));
      try {
        const { data } = await requestJSON("/api/hermes/install", {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind: kind === "plugins" ? "plugin" : "skill", name: kind === "plugins" ? item.name : item.identifier }),
        });
        action.replaceChildren(text("small", data.ok ? "Installed, switched off" : "Did not install"));
        note(data.ok ? `${item.name} is installed but switched off. Enable it in Hermes, then restart Hermes.` : data.message || "The install did not work");
      } catch (error) {
        action.replaceChildren(text("small", "Did not install"));
        note(error.message || "The install did not work");
      }
    };
    action.replaceChildren(ask, yes, no);
  }

  return {
    bind() {
      const search = async () => {
        const kind = $("#hermes-catalog-kind").value, box = $("#hermes-catalog"), go = $("#hermes-catalog-go");
        go.disabled = true;
        note("Searching…");
        try {
          const { data } = await requestJSON(`/api/hermes/catalog?kind=${kind}&q=${encodeURIComponent($("#hermes-catalog-q").value.trim())}`);
          if (!data.available) { note(data.reason || "Not available"); box.hidden = true; return; }
          const items = data.result.items || [];
          box.replaceChildren(...items.map((item) => row(kind, item, box)));
          box.hidden = !items.length;
          note(items.length ? `${items.length} found. Official and verified ones can be installed here.` : kind === "skills" && $("#hermes-catalog-q").value.trim().length < 2 ? "Type at least two letters to search skills" : "Nothing found");
        } catch (error) {
          note(error.message || "The search did not work");
        } finally {
          go.disabled = false;
        }
      };
      $("#hermes-catalog-go").onclick = search;
      $("#hermes-catalog-q").addEventListener("keydown", (event) => { if (event.key === "Enter") { event.preventDefault(); search(); } });
    },
  };
}
