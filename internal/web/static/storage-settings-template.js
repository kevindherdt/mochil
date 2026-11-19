export const storageSettingsHTML = `
    <details class="settings-accordion storage-settings" id="scan-settings">
      <summary><span><strong>Advanced</strong><small>Default scan interval and cover cache</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body">
        <section class="settings-group settings-fields" aria-labelledby="scan-group">
          <h3 id="scan-group">Scanning and covers</h3>
          <label>Default scan interval<input id="scan-every" value="10m" aria-describedby="scan-help" /></label><small id="scan-help">Examples: 10m, 1h, or 0 to disable. A library can set its own under More options.</small>
          <label>Cover cache (MiB)<input id="cover-cache-mib" type="number" min="16" max="102400" step="16" value="512" aria-describedby="cover-cache-help" /></label><small id="cover-cache-help">Resized covers; oldest unused files are removed first.</small>
          <div><button type="button" id="scan-all-settings">Scan all libraries now</button></div>
        </section>
      </div>
    </details>`;

export function mountStorageSettings(document) {
  if (document.querySelector("#scan-settings")) return;
  document
    .querySelector('[data-settings-panel="libraries"]')
    .insertAdjacentHTML("beforeend", storageSettingsHTML);
}
