export const activityPanelHTML = `
  <header>
    <div><strong>Activity</strong><small>Library scans</small></div>
    <button type="button" id="scan-all-libraries">Scan all</button>
  </header>
  <p class="activity-message" id="activity-message" role="status"></p>
  <div id="activity-list"></div>`;

export function mountActivityPanel(document) {
  if (document.querySelector("#activity-panel")) return;
  const panel = document.createElement("aside");
  panel.className = "activity-panel";
  panel.id = "activity-panel";
  panel.hidden = true;
  panel.setAttribute("aria-label", "Library activity");
  panel.innerHTML = activityPanelHTML;
  document.querySelector("nav").insertAdjacentElement("afterend", panel);
}
