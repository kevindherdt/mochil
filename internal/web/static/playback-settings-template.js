export const playbackSettingsHTML = `
  <section class="settings-panel" data-settings-panel="playback" hidden>
    <header><h2>Playback</h2><p>Languages and streaming quality.</p></header>
    <div class="system-card settings-fields">
      <label>Wi-Fi quality<select id="playback-wifi-quality"><option value="maximum">Maximum · original when supported</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option></select></label>
      <small>Maximum avoids conversion whenever the device can play the source.</small>
      <label>Mobile-data quality<select id="playback-mobile-quality"><option value="maximum">Maximum</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option></select></label>
      <small>Clients use this preference when Android reports a metered connection.</small>
      <label>Default audio language<input id="playback-audio-language" list="media-languages" placeholder="File default" /></label>
      <small>Leave blank to use the audio marked as default in each file.</small>
      <label>Default subtitles<input id="playback-subtitle-language" list="subtitle-languages" placeholder="Off" /></label>
      <small>Use <strong>off</strong> for no subtitles, or enter a language such as English, Japanese, or Korean.</small>
      <datalist id="media-languages"><option value="English"></option><option value="Japanese"></option><option value="Korean"></option><option value="Dutch"></option><option value="French"></option><option value="German"></option><option value="Spanish"></option><option value="Chinese"></option></datalist>
      <datalist id="subtitle-languages"><option value="off"></option><option value="English"></option><option value="Japanese"></option><option value="Korean"></option><option value="Dutch"></option><option value="French"></option><option value="German"></option><option value="Spanish"></option><option value="Chinese"></option></datalist>
    </div>
  </section>`;

export function mountPlaybackSettings(document) {
  if (document.querySelector('[data-settings-panel="playback"]')) return;
  document
    .querySelector("#settings-view")
    .insertAdjacentHTML("beforeend", playbackSettingsHTML);
}
