export const storageSettingsHTML = `
    <details class="settings-accordion storage-settings" id="scan-settings">
      <summary><span><strong>Advanced</strong><small>Default scan interval, cover cache and managed ingest</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body">
        <section class="settings-group settings-fields" aria-labelledby="scan-group">
          <h3 id="scan-group">Scanning and covers</h3>
          <label>Default scan interval<input id="scan-every" value="10m" aria-describedby="scan-help" /></label><small id="scan-help">Examples: 10m, 1h, or 0 to disable. A library can set its own under More options.</small>
          <label>Cover cache (MiB)<input id="cover-cache-mib" type="number" min="16" max="102400" step="16" value="512" aria-describedby="cover-cache-help" /></label><small id="cover-cache-help">Resized covers; oldest unused files are removed first.</small>
          <div><button type="button" id="scan-all-settings">Scan all libraries now</button></div>
        </section>
        <section class="settings-group settings-fields" id="managed-ingest-settings" aria-labelledby="ingest-group">
          <h3 id="ingest-group">Managed ingest</h3>
          <p>Opt-in. Disabled until both folders are set. Mochil copies verified files; it never deletes incoming files or replaces managed files.</p>
          <label>Incoming folder<input id="managed-ingest-incoming" placeholder="/media/incoming" autocomplete="off" aria-describedby="managed-ingest-help" /></label><small id="managed-ingest-help">A dedicated drop folder, separate from every managed library.</small>
          <label>Managed folder<input id="managed-ingest-target" placeholder="/media/library" autocomplete="off" aria-describedby="managed-ingest-target-help" /></label><small id="managed-ingest-target-help">Verified copies are committed below this folder without overwriting existing paths.</small>
          <label>Maximum file size (GiB)<input id="managed-ingest-gib" type="number" min="1" max="1024" step="1" value="8" aria-describedby="managed-ingest-limit-help" /></label><small id="managed-ingest-limit-help">The incoming and managed folders cannot overlap.</small>
        </section>
      </div>
    </details>`;

export function mountStorageSettings(document) {
  if (document.querySelector("#scan-settings")) return;
  document
    .querySelector('[data-settings-panel="libraries"]')
    .insertAdjacentHTML("beforeend", storageSettingsHTML);
}
