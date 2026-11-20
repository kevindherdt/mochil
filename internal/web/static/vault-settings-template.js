export const vaultSettingsHTML = `
      <label>Wiki page width<select id="vault-width"><option value="centered">Readable line length</option><option value="wide">Wide, aligned left</option></select></label>
      <small>Readable keeps long lines comfortable. Wide uses the available screen and aligns the page to the left.</small>`;

export function mountVaultSettings(document) {
  if (document.querySelector("#vault-width")) return;
  document
    .querySelector('[data-settings-panel="appearance"] .appearance-settings')
    .insertAdjacentHTML("beforeend", vaultSettingsHTML);
}
