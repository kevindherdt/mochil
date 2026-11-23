// Settings → Modules → Diagnostics: a read-only health summary so a slow or
// failing server can be understood without reading logs.
const escapeHTML = (value) =>
  String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]);

export function formatUptime(seconds) {
  const total = Math.max(0, Math.floor(seconds));
  const days = Math.floor(total / 86400), hours = Math.floor((total % 86400) / 3600), minutes = Math.floor((total % 3600) / 60);
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function rows(list, empty) {
  if (!list.length) return `<p class="diagnostics-empty">${empty}</p>`;
  return `<ul class="diagnostics-list">${list
    .map((item) => `<li><span>${escapeHTML(item.method)} ${escapeHTML(item.path)}</span><small>${item.status} · ${item.duration_ms} ms · ${escapeHTML(new Date(item.at).toLocaleTimeString())}</small></li>`)
    .join("")}</ul>`;
}

function background(list) {
  if (!list.length) return `<p class="diagnostics-empty">Nothing in the background has failed since the server started.</p>`;
  return `<ul class="diagnostics-list">${list
    .map((item) => `<li><span>${escapeHTML(item.what)}</span><small>${escapeHTML(ago(item.at))}${item.count > 1 ? ` · ${item.count} times` : ""}${item.restarts ? " · restarted" : ""} · ${escapeHTML(item.message)}</small></li>`)
    .join("")}</ul>`;
}

export function ago(at, now = Date.now()) {
  const minutes = Math.max(0, Math.floor((now - at) / 60000));
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  return hours < 48 ? `${hours} h ago` : `${Math.floor(hours / 24)} d ago`;
}

export function diagnosticsHTML(data) {
  const stat = (label, value) => `<div><strong>${escapeHTML(value)}</strong><span>${label}</span></div>`;
  return `
    <div class="diagnostics-stats">
      ${stat("Uptime", formatUptime(data.uptime_seconds))}
      ${stat("Memory in use", `${Number(data.heap_mb).toFixed(0)} MB`)}
      ${stat("Requests", Number(data.requests).toLocaleString())}
      ${stat("Server errors", Number(data.failures).toLocaleString())}
      ${stat("Libraries", data.libraries)}
      ${stat("Goroutines", data.goroutines)}
    </div>
    <h3>Slowest requests</h3>${rows(data.slow || [], "Nothing has been slow since the server started.")}
    <h3>Background work that failed</h3>${background(data.background || [])}
    <h3>Recent server errors</h3>${rows(data.errors || [], "No server errors since the server started.")}`;
}

// The running build, in the same stat blocks the panel already uses. This exists because "which version are you on?"
// used to be unanswerable from inside the app, and every mismatch between a phone showing old markup and a server
// serving new markup turned into guesswork about caches. Now it is a number on screen.
export function versionHTML(info) {
  const stat = (label, value) => `<div><strong>${escapeHTML(value)}</strong><span>${label}</span></div>`;
  const build = info.version === "0.0.0-dev" ? "development build" : `Mochil ${info.version}`;
  return `
    <div class="diagnostics-stats">
      ${stat("Version", build)}
      ${stat("Commit", info.commit || "unknown")}
      ${stat("Started", info.started ? new Date(info.started).toLocaleString() : "unknown")}
      ${stat("Uptime", info.uptime || "0s")}
    </div>`;
}

export function mountDiagnostics(document, fetchJSON = (url) => fetch(url, { cache: "no-store" }).then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))) {
  const panel = document.querySelector('[data-settings-panel="diagnostics"]');
  if (!panel || panel.querySelector("#diagnostics")) return;
  panel.insertAdjacentHTML("beforeend", `<details class="settings-accordion" id="diagnostics"><summary><span><strong>Diagnostics</strong><small>Server health at a glance</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary><div class="settings-accordion-body" id="diagnostics-body" aria-live="polite"></div></details>`);
  const details = panel.querySelector("#diagnostics");
  const body = panel.querySelector("#diagnostics-body");
  const load = async () => {
    body.textContent = "Loading…";
    try {
      body.innerHTML = diagnosticsHTML(await fetchJSON("/api/diagnostics"));
    } catch {
      body.textContent = "Diagnostics are unavailable right now.";
    }
  };
  details.addEventListener("toggle", () => (details.open ? load() : undefined));
  panel.insertAdjacentHTML("beforeend", `<details class="settings-accordion" id="about" open><summary><span><strong>About</strong><small>Which build this is</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary><div class="settings-accordion-body" id="about-body"></div></details>`);
  const aboutBody = panel.querySelector("#about-body");
  const loadAbout = async () => {
    if (!aboutBody) return;
    try {
      aboutBody.innerHTML = versionHTML(await fetchJSON("/api/version"));
    } catch {
      aboutBody.textContent = "The version is unavailable right now.";
    }
  };
  loadAbout();
  document.querySelector('[data-settings-tab="diagnostics"]')?.addEventListener?.("click", loadAbout);
  // Its own tab: opening the tab opens the section and refreshes the numbers.
  document.querySelector('[data-settings-tab="diagnostics"]')?.addEventListener?.("click", () => (details.open ? load() : (details.open = true)));
}
