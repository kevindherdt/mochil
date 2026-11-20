export const themeSettingsHTML = `
  <section class="settings-panel" data-settings-panel="appearance" hidden>
    <header><h2>Appearance</h2><p>How Mochil looks and reads. These choices are saved in this browser only.</p></header>
    <div class="system-card appearance-settings">
      <label>Theme family<select id="theme-palette"><option value="mochil">Ember</option><option value="kiln">Tide</option><option value="magenta-cream">Magenta Cream (bold)</option><option value="eink">e-ink</option></select></label>
      <small>Ember is warm, Tide is cool; both follow light and dark. Magenta Cream is the mark's own colours. Paper keeps one brightness. E-ink removes shadows and uses maximum contrast.</small>
      <label>Brightness<select id="theme-mode"><option value="system">Follow device</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
      <label class="check-field"><input type="checkbox" id="reduce-motion"><span><strong>Reduce animations</strong><small>Panels, lists and page changes happen at once instead of sliding or fading. On by default with the E-ink theme and when your device asks for less motion.</small></span></label>
    </div>
  </section>`;

export function mountThemeSettings(document) {
  if (document.querySelector('[data-settings-panel="appearance"]')) return;
  document
    .querySelector("#settings-view")
    .insertAdjacentHTML("beforeend", themeSettingsHTML);
}
