import { mountActivityPanel } from './activity-panel-template.js?v=1'

// How long to wait before asking about scans again: quickly while one runs, rarely when nobody has the panel open.
export function refreshDelay(jobs, panelHidden) {
  if (jobs.some((job) => ["queued", "running"].includes(job.status))) return 1000;
  return panelHidden ? 15000 : 5000;
}

export function createActivityMonitor({ document, $, escapeHTML, requestJSON }) {
  mountActivityPanel(document)
  const button = $("#activity-toggle"),
    panel = $("#activity-panel"),
    list = $("#activity-list"),
    message = $("#activity-message"),
    scanAll = $("#scan-all-libraries");
  let timer,
    jobs = [];
  const libraryName = (key) =>
    (key.split(":")[1] || "library").replaceAll("-", " ");
  const time = (value) =>
    value
      ? new Date(value).toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
      : "";
  const render = () => {
    const active = jobs.some((job) =>
      ["queued", "running"].includes(job.status),
    );
    const failed = jobs.some((job) => job.status === "failed");
    // Quiet by default: the icon appears only while a scan runs, after one fails,
    // or while its panel is open.
    button.hidden = !(active || failed || !panel.hidden);
    button.classList.toggle("active", active);
    button.setAttribute(
      "aria-label",
      active ? "Library scan in progress" : "Activity",
    );
    button.setAttribute("aria-expanded", String(!panel.hidden));
    list.innerHTML = jobs.length
      ? jobs
          .slice(0, 6)
          .map((job) => {
            const current = Number(job.progress_current || 0),
              total = Number(job.progress_total || 0),
              percent =
                total > 0
                  ? Math.min(100, Math.round((current / total) * 100))
                  : 0;
            const label =
              job.status === "ready"
                ? "Complete"
                : job.status === "failed"
                  ? "Failed"
                  : job.status === "queued"
                    ? "Waiting"
                    : "Scanning";
            const detail =
              job.error ||
              job.progress_message ||
              (job.status === "queued"
                ? "Waiting for an available scanner…"
                : "");
            const count =
              total > 0 && job.status !== "queued"
                ? `${current.toLocaleString()} / ${total.toLocaleString()}`
                : "";
            return `<article class="activity-row" data-state="${escapeHTML(job.status)}"><span class="activity-state" aria-hidden="true"></span><div><strong>${escapeHTML(libraryName(job.key))}</strong><small>${escapeHTML(label)}${count ? ` · ${escapeHTML(count)}` : ""}${time(job.started_at || job.queued_at) ? ` · ${escapeHTML(time(job.started_at || job.queued_at))}` : ""}</small>${detail ? `<p title="${escapeHTML(detail)}">${escapeHTML(detail)}</p>` : ""}${total > 0 ? `<progress value="${current}" max="${total}" aria-label="${percent}% scanned"></progress>` : ""}</div></article>`;
          })
          .join("")
      : '<p class="activity-empty">No library scans yet.</p>';
  };
  async function refresh() {
    try {
      const response = await requestJSON("/api/jobs");
      jobs = (response.data || [])
        .filter((job) => job.queue === "library-scans")
        .sort((a, b) => String(b.queued_at).localeCompare(String(a.queued_at)));
      render();
    } catch (_) {}
    clearTimeout(timer);
    if (document.hidden) return; // nobody is looking; coming back refreshes at once
    timer = window.setTimeout(refresh, refreshDelay(jobs, panel.hidden));
  }
  button.onclick = () => {
    panel.hidden = !panel.hidden;
    render();
    if (!panel.hidden) refresh();
  };
  scanAll.onclick = async () => {
    scanAll.disabled = true;
    message.textContent = "Queueing libraries…";
    try {
      await requestJSON("/api/library-scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ all: true }),
      });
      message.textContent = "All libraries queued.";
      await refresh();
    } catch (error) {
      message.textContent =
        error.status === 401
          ? "Your session expired. Reload and sign in before scanning."
          : error.message;
    } finally {
      scanAll.disabled = false;
    }
  };
  document.addEventListener("click", (event) => {
    if (event.target.closest?.("#scan-all-settings")) {
      panel.hidden = false;
      scanAll.onclick();
      return;
    }
    if (
      !panel.hidden &&
      !panel.contains(event.target) &&
      !button.contains(event.target)
    ) {
      panel.hidden = true;
      render();
    }
  });
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) refresh();
  });
  refresh();
  return { refresh };
}
