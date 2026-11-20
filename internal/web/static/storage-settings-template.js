export const storageSettingsHTML = `
    <details class="settings-accordion storage-settings" id="scan-settings">
      <summary><span><strong>Scanning</strong><small>Re-read every library now</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body">
        <p>Mochil watches for new files on its own. Use this when you have copied a batch in and do not want to wait: each library picks up new books at its own interval, which you can change per library under its own settings.</p>
        <div><button type="button" id="scan-all-settings">Scan all libraries now</button></div>
      </div>
    </details>`;

export function mountStorageSettings(document) {
  if (document.querySelector("#scan-settings")) return;
  document
    .querySelector('[data-settings-panel="libraries"]')
    .insertAdjacentHTML("beforeend", storageSettingsHTML);
}
