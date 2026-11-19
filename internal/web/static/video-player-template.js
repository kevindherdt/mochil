export const videoPlayerHTML = `
  <header>
    <div class="video-heading">
      <strong id="video-title"></strong><small><span id="video-runtime"></span><span id="video-playback-mode"></span></small>
    </div>
    <div class="video-player-actions">
      <details class="track-menu" id="track-menu" hidden>
        <summary aria-label="Choose audio and subtitles" title="Audio and subtitles">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 10v4M8 7v10M12 5v14M16 8h4M16 12h4M16 16h4" /></svg>
        </summary>
        <div>
          <label id="audio-track-field">Audio<select id="audio-track"></select></label>
          <label id="subtitle-track-field">Subtitles<select id="subtitle-track"></select></label>
        </div>
      </details>
      <button id="video-close" aria-label="Close video">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg>
      </button>
    </div>
  </header>
  <div class="video-stage">
    <video id="video-element" playsinline preload="metadata"></video>
    <figure class="video-preview" id="video-preview" hidden aria-hidden="true">
      <span id="video-preview-image"></span>
      <figcaption id="video-preview-time"></figcaption>
    </figure>
    <button class="video-centre-play" id="video-centre-play" aria-label="Play"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 5 12 7-12 7Z" /></svg></button>
    <div class="video-controls">
      <input class="video-seek" id="video-seek" type="range" min="0" max="1000" value="0" aria-label="Playback position" />
      <div class="video-controls-row">
        <button id="video-play" aria-label="Play"><svg class="v-play" viewBox="0 0 24 24" aria-hidden="true"><path d="m8 5 12 7-12 7Z" /></svg><svg class="v-pause" viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5v14M16 5v14" /></svg></button>
        <button id="video-back" aria-label="Back 10 seconds"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 12a8 8 0 1 0 3-6.2M4 4v4h4" /><text x="12" y="15.5" text-anchor="middle">10</text></svg></button>
        <button id="video-forward" aria-label="Forward 10 seconds"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 12a8 8 0 1 1-3-6.2M20 4v4h-4" /><text x="12" y="15.5" text-anchor="middle">10</text></svg></button>
        <span class="video-time"><span id="video-elapsed">0:00</span><i>/</i><span id="video-total"></span></span>
        <span class="video-volume"><button id="video-mute" aria-label="Mute"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9H4Z" /><path class="wave" d="M16.5 9a4 4 0 0 1 0 6M19 6.5a8 8 0 0 1 0 11" /><path class="slash" d="m16 9 5 6m0-6-5 6" /></svg></button><input id="video-volume" type="range" min="0" max="100" value="100" aria-label="Volume" /></span>
        <button id="video-fullscreen" aria-label="Full screen"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5" /></svg></button>
      </div>
    </div>
  </div>
  <p id="video-message" role="status"></p>`;

export function mountVideoPlayer(document) {
  if (document.getElementById("video-player")) return;
  const player = document.createElement("dialog");
  player.className = "video-player";
  player.id = "video-player";
  player.innerHTML = videoPlayerHTML;
  document.body.append(player);
}
