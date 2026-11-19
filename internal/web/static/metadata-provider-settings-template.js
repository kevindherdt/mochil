export const metadataProviderSettingsHTML = `
  <details class="settings-accordion provider-connections" data-metadata-provider-settings>
    <summary><span><strong>Metadata providers</strong><small>Optional credentials for online suggestions</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
    <div class="settings-accordion-body provider-fields">
      <p>Online lookup sends a title and, when available, its author only after you request suggestions. Mochil never uploads media files.</p>
      <section class="provider-field" data-provider="hardcover">
        <label>Hardcover API token<input id="hardcover-token" type="password" autocomplete="new-password" maxlength="4096" placeholder="Not configured" aria-describedby="hardcover-provider-status" /></label>
        <div class="provider-field-footer"><small id="hardcover-provider-status" role="status">For publication metadata.</small><button type="button" data-test-provider="hardcover">Test connection</button></div>
      </section>
      <section class="provider-field" data-provider="tmdb">
        <label>TMDB API Read Access Token<input id="tmdb-token" type="password" autocomplete="new-password" maxlength="4096" placeholder="Not configured" aria-describedby="tmdb-provider-status" /></label>
        <div class="provider-field-footer"><small id="tmdb-provider-status" role="status">For movie and television metadata.</small><button type="button" data-test-provider="tmdb">Test connection</button></div>
      </section>
      <small>Leave a token blank to keep the saved value. Testing a pasted token does not save it.</small>
    </div>
  </details>`;

export function mountMetadataProviderSettings(document) {
  if (document.querySelector("[data-metadata-provider-settings]")) return;
  document
    .querySelector('[data-settings-panel="metadata"]')
    .insertAdjacentHTML("beforeend", metadataProviderSettingsHTML);
}
