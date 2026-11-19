// What she remembers. The memories live in Hindsight (a separate server Hermes writes to after each turn); the page
// only lists them, read-only, and saves a message on request through "Remember". Everything goes through /api/hermes.

const ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 11c0 5.6-7 10-7 10z"/></svg>';

export function relativeDate(value, now = Date.now()) {
  const time = Date.parse(value);
  if (!Number.isFinite(time)) return "";
  const days = Math.floor((now - time) / 86400000);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 30) return `${days} days ago`;
  return new Date(time).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
}

export function createMemory({ document, $, api, name = () => "she", onOpen = () => {} }) {
  let enabled = false;
  const open = $("#hm-mem-open");
  const panel = $("#hm-memory");
  const list = $("#hm-mem-list");
  const note = $("#hm-mem-note");

  async function load() {
    note.textContent = "Loading…";
    list.replaceChildren();
    try {
      const data = await api("/api/hermes/memories");
      const items = data.items || [];
      note.textContent = items.length
        ? `${items.length} thing${items.length === 1 ? "" : "s"} ${name()} has picked up from your chats. She keeps them herself; this list is only for looking.`
        : `${name()} has not remembered anything yet. Talk for a while, or use Remember on a message. New memories are written when her model is on.`;
      for (const item of items) {
        const row = document.createElement("li");
        const text = document.createElement("span");
        text.textContent = item.text;
        const when = document.createElement("small");
        when.textContent = relativeDate(item.date);
        row.append(text, when);
        list.append(row);
      }
    } catch (error) {
      note.textContent = error.message || "Could not load her memories";
    }
  }

  function show(on) {
    panel.hidden = !on;
    open.setAttribute("aria-expanded", String(on));
    if (on) load();
  }

  open.onclick = () => { const on = panel.hidden; show(on); if (on) onOpen(); };
  $("#hm-mem-close").onclick = () => show(false);
  $("#hm-mem-refresh").onclick = load;

  return {
    setEnabled(value) { enabled = Boolean(value); open.hidden = !enabled; if (!enabled) show(false); },
    get enabled() { return enabled; },
    /** A small button that saves text as a memory and says what happened. */
    button(getText) {
      const button = document.createElement("button");
      button.type = "button"; button.className = "hm-tool"; button.innerHTML = `${ICON}Remember`;
      button.onclick = async () => {
        const text = String(getText() || "").trim();
        if (!text || button.disabled) return;
        button.disabled = true;
        try {
          await api("/api/hermes/remember", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text }) });
          button.lastChild.textContent = "Sent to memory";
          if (!panel.hidden) setTimeout(load, 2500);
        } catch (_) {
          button.lastChild.textContent = "Could not save";
          button.disabled = false;
          setTimeout(() => { button.lastChild.textContent = "Remember"; }, 2500);
        }
      };
      return button;
    },
  };
}
