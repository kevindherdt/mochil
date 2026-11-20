export const readingSyncSettingsHTML = `
  <section class="settings-panel" data-settings-panel="sync" hidden>
    <header><h2>Devices &amp; sync</h2><p>Keep your browser, KOReader, and apps on the same page.</p></header>
    <details class="settings-accordion sync-settings" open>
      <summary><span><strong>Position sync</strong><small>Provider, account, and cross-reader accuracy</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body system-card">
        <label>Sync provider<select id="kosync-provider" aria-describedby="kosync-provider-help"><option value="built-in">Built into Mochil</option><option value="external">External KoSync server</option><option value="disabled">Disabled</option></select></label>
        <small id="kosync-provider-help">Choose the service shared with KOReader.</small>
        <label class="setting-toggle"><input id="kosync-webreader-bridge" type="checkbox" /><span><strong>Cross-reader precision sync</strong><small>Share text-anchored positions between KOReader, Android, and the web reader. KoSync percentages remain the fallback.</small></span></label>
        <div id="kosync-external-fields" hidden><label>External server URL<input id="kosync-server-url" type="url" placeholder="https://sync.koreader.rocks" aria-describedby="kosync-url-help" /></label><small id="kosync-url-help">Enter the same base URL in KOReader.</small></div>
        <p class="sync-address" id="kosync-local-help">Use this Mochil address in KOReader.</p>
        <small id="plugin-server-note" hidden></small>
        <div class="kosync-credentials" id="kosync-credentials" hidden>
          <label>Sync username<input id="kosync-username" autocomplete="username" aria-describedby="kosync-account-help" /></label>
          <label>Sync password<input id="kosync-password" type="password" autocomplete="new-password" aria-describedby="kosync-account-help" /></label>
          <small id="kosync-account-help">Leave a saved password blank to keep it unchanged.</small>
        </div>
      </div>
    </details>
    <details class="settings-accordion sync-settings" id="plugin-setup">
      <summary><span><strong>KOReader plugin</strong><small>Optional paragraph-level precision for KOReader</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body plugin-body">
        <button class="primary" id="download-plugin">Download paired plugin</button>
        <ol><li>Download it on your reader.</li><li>Extract it into <strong>koreader/plugins</strong>.</li><li>Restart KOReader.</li></ol>
        <small>Each paired device remains connected when you create another package.</small>
        <p id="plugin-status" role="status"></p>
      </div>
    </details>
    <details class="settings-accordion sync-settings" id="android-setup">
      <summary><span><strong>Android and TV app</strong><small>Install or update the native Mochil client</small></span><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg></summary>
      <div class="settings-accordion-body plugin-body">
        <a class="primary download-action" id="download-android" href="/api/android/app.apk" download="mochil-android.apk">Download Android app</a>
        <ol><li>Download the APK on your Android device.</li><li>Allow your browser or file manager under <strong>Settings → Apps → Special app access → Install unknown apps</strong>, then open the APK again.</li><li>On Google TV, enable Developer options first if that permission is hidden: press <strong>Android TV OS build</strong> seven times under Settings → System → About.</li><li>Enter this Mochil address and sign in.</li></ol>
        <small id="android-version">Checking the available build…</small>
        <p id="android-status" role="status"></p>
      </div>
    </details>
  </section>`;

export function mountReadingSyncSettings(document) {
  if (document.querySelector('[data-settings-panel="sync"]')) return;
  document
    .querySelector("#settings-view")
    .insertAdjacentHTML("beforeend", readingSyncSettingsHTML);
}
