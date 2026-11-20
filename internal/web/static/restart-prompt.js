// Some settings (a module switched on or off) only take effect when Mochil starts again. This puts a
// "Restart now" button beside the save message, so the person does not have to restart it some other way.
export function offerRestart({ document, status, fetchImpl = fetch, wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms)), reload = () => location.reload(), tries = 60 }) {
  status.textContent = "Saved. This change takes effect after a restart.";
  let button = document.querySelector("#restart-now");
  if (!button) {
    button = document.createElement("button");
    button.type = "button";
    button.id = "restart-now";
    button.className = "restart-now";
    status.after(button);
  }
  button.textContent = "Restart now";
  button.disabled = false;
  button.hidden = false;
  button.onclick = async () => {
    button.disabled = true;
    button.textContent = "Restarting…";
    try {
      const reply = await fetchImpl("/api/restart", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
      if (!reply.ok) throw new Error(`HTTP ${reply.status}`);
    } catch (error) {
      button.disabled = false;
      button.textContent = "Restart now";
      status.textContent = `Could not restart: ${error.message}`;
      return;
    }
    for (let attempt = 0; attempt < tries; attempt++) {
      await wait(2000);
      try { if ((await fetchImpl("/api/libraries")).ok) { reload(); return; } } catch (_) { /* still starting */ }
    }
    button.hidden = true;
    status.textContent = "Restarting is taking long. Reload this page in a moment.";
  };
  return button;
}

// A later save that needs no restart must withdraw the earlier offer: the button is created once and
// stayed live, so the page went on asking for a restart nothing was waiting for.
export function withdrawRestart(document) {
  const button = document.querySelector("#restart-now");
  if (!button) return;
  button.hidden = true;
  button.disabled = true;
  button.onclick = null;
}
