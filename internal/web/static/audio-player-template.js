export const audioPlayerHTML = `
  <audio id="audio-element" preload="metadata" hidden></audio>
  <audio id="audio-gapless-element" preload="auto" hidden></audio>
  <input class="audio-seek" id="audio-seek" type="range" min="0" max="1000" value="0" aria-label="Playback position" />
  <span class="audio-vinyl" aria-hidden="true"></span>
  <span class="audio-art-play" id="audio-art">
    <span id="audio-player-art"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 18V5l11-2v13M9 9l11-2" /></svg></span>
  </span>
  <button class="audio-player-copy" id="audio-expand" aria-label="Expand player" aria-expanded="false">
    <strong id="audio-title"></strong><small id="audio-context"></small><span id="audio-source-facts"></span>
  </button>
  <span class="audio-bar-skip">
    <button class="audio-bar-extra" id="audio-bar-shuffle" aria-label="Enable shuffle" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h3c5 0 5 10 10 10h3m-4-3 4 3-4 3M4 17h3c2 0 3-1.5 4-3m3-4c1-2 2-3 3-3h3m-4-3 4 3-4 3" /></svg></button>
    <button id="audio-bar-previous" aria-label="Previous"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 5v14m12-13-9 6 9 6Z" /></svg></button>
    <button class="audio-bar-play" id="audio-play" aria-label="Play"><svg class="audio-play-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="m9 7 9 5-9 5Z" /></svg></button>
    <button id="audio-bar-next" aria-label="Next"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 5v14M6 6l9 6-9 6Z" /></svg></button>
    <button class="audio-bar-extra" id="audio-bar-repeat" data-mode="off" aria-label="Repeat off" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m17 3 4 4-4 4M3 11V9a2 2 0 0 1 2-2h16M7 21l-4-4 4-4M21 13v2a2 2 0 0 1-2 2H3" /></svg></button>
    <button id="audio-bar-close" aria-label="Close player"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg></button>
  </span>
  <button class="audio-retry" id="audio-retry" hidden>Try again</button>
  <span class="audio-time"><span id="audio-elapsed">0:00</span><i>/</i><span id="audio-duration">0:00</span></span>
  <div class="audio-expanded-transport">
    <button id="audio-player-shuffle-large" aria-label="Enable shuffle" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h3c5 0 5 10 10 10h3m-4-3 4 3-4 3M4 17h3c2 0 3-1.5 4-3m3-4c1-2 2-3 3-3h3m-4-3 4 3-4 3" /></svg></button>
    <button id="audio-previous-large" aria-label="Previous"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 5v14m12-13-9 6 9 6Z" /></svg></button>
    <button id="audio-play-large" aria-label="Play"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 7 9 5-9 5Z" /></svg></button>
    <button id="audio-next-large" aria-label="Next"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 5v14M6 6l9 6-9 6Z" /></svg></button>
    <button id="audio-queue-toggle" aria-label="Open queue" aria-expanded="false" aria-controls="audio-player-queue"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 7h10M5 12h10M5 17h7M18 15v5m-2.5-2.5L18 20l2.5-2.5" /></svg></button>
  </div>
  <section class="audio-player-queue" id="audio-player-queue" aria-label="Playback queue" hidden>
    <header><span><strong>Queue</strong><small id="audio-queue-position"></small></span><button id="audio-queue-close" aria-label="Close queue"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg></button></header>
    <div id="audio-queue-items"></div>
    <footer><button id="audio-queue-earlier">Earlier</button><button id="audio-queue-later">Later</button></footer>
  </section>
  <section class="audio-chapters" id="audio-chapters" aria-label="Chapters" hidden>
    <header><strong>Chapters</strong><button id="audio-chapters-close" aria-label="Close chapters"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg></button></header>
    <div id="audio-chapter-list"></div>
  </section>
  <button class="audio-player-collapse" id="audio-collapse" hidden aria-label="Collapse player"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg></button>
  <button class="audio-player-favorite" id="audio-player-favorite" hidden aria-label="Add to favorites" aria-pressed="false"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20.5 4.5 13A5 5 0 0 1 12 6.5 5 5 0 0 1 19.5 13Z" /></svg></button>
  <span class="audio-volume">
    <button id="audio-mute" aria-label="Mute"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9H4Z" /><path class="wave" d="M16.5 9a4 4 0 0 1 0 6M19 6.5a8 8 0 0 1 0 11" /><path class="slash" d="m16 9 5 6m0-6-5 6" /></svg></button>
    <input id="audio-volume" type="range" min="0" max="100" value="100" aria-label="Volume" />
  </span>
  <details class="audio-player-more">
    <summary aria-label="More playback options"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="5" r="1.4" /><circle cx="12" cy="12" r="1.4" /><circle cx="12" cy="19" r="1.4" /></svg></summary>
    <div>
      <div class="audio-player-skip">
        <button id="audio-previous"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 5v14m12-13-9 6 9 6Z" /></svg><span>Previous</span></button>
        <button id="audio-next"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 5v14M6 6l9 6-9 6Z" /></svg><span>Next</span></button>
      </div>
      <label class="audio-speed"><span>Speed</span><select id="audio-speed" aria-label="Playback speed">
        <option value="0.75">0.75×</option><option value="1" selected>1×</option><option value="1.25">1.25×</option><option value="1.5">1.5×</option><option value="1.75">1.75×</option><option value="2">2×</option>
      </select></label>
      <label class="audio-speed"><span>ReplayGain</span><select id="audio-replaygain" aria-label="ReplayGain mode">
        <option value="track">Track</option><option value="album">Album</option><option value="off">Off</option>
      </select></label>
      <button id="audio-chapters-toggle" hidden aria-expanded="false" aria-controls="audio-chapters"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 4h14v16H5zM9 8h6M9 12h6M9 16h4"/></svg><span>Chapters</span></button>
      <button id="audio-sleep"><span>Sleep timer</span><b>Zz</b></button>
      <button id="audio-close"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17" /></svg><span>Close player</span></button>
    </div>
  </details>`;

export function mountAudioPlayer(document) {
  if (document.getElementById("audio-player")) return;
  const player = document.createElement("section");
  player.className = "audio-player";
  player.id = "audio-player";
  player.hidden = true;
  player.setAttribute("aria-label", "Audio player");
  player.innerHTML = audioPlayerHTML;
  document.body.append(player);
}
