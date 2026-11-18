import { createVideoProgress } from "./video-progress.js?v=4";
import { loadHLSClient } from "./hls-client.js?v=1";
import { mountVideoPlayer } from "./video-player-template.js?v=2";
import { effectivePlaybackQuality, qualityHLSURL } from "./playback-quality.js?v=2";
import { createVideoControls } from "./video-controls.js?v=4";
import { createMediaSeekOrigin } from "./media-seek-origin.js?v=1";
import {
  preferredVideoTrack,
  videoTimeLabel as timeLabel,
  videoTrackLabel as trackLabel,
} from "./video-format.js?v=2";

export function videoHLSURL(item, audioIndex, quality = 'high') {
  return `${qualityHLSURL(`/api/video-hls/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}/${encodeURIComponent(audioIndex)}/index.m3u8`, quality)}?v=${encodeURIComponent(item.revision || "current")}-7`;
}

export function videoResumePosition(duration, requested, finished = false) {
  const knownDuration = Number(duration);
  const position = Number(requested);
  return !finished && Number.isFinite(position) &&
    position > 0 &&
    (Number.isNaN(knownDuration) || position < knownDuration - 5)
    ? position
    : 0;
}

// Catalogue objects are snapshots; drain local writes before refreshing one.
export function createVideoOpenProgress(requestJSON, { timeoutMs = 8000 } = {}) {
  const pending = new Map(), failures = new Map();
  let controller, serial = 0;
  const key = item => JSON.stringify([item.library_id, item.id]);
  const cancel = () => { controller?.abort(); controller = null; };
  const wait = (promise, signal, deadline) => new Promise((resolve, reject) => {
    const timer = setTimeout(() => finish(reject, new Error('Video progress request timed out. Please try again.')), Math.max(0, deadline - Date.now()));
    const abort = () => finish(resolve, null);
    function finish(done, value) {
      clearTimeout(timer);
      signal.removeEventListener('abort', abort);
      done(value);
    }
    signal.addEventListener('abort', abort, { once: true });
    Promise.resolve(promise).then(value => finish(resolve, value), error => finish(reject, error));
    if (signal.aborted) abort();
  });
  return {
    track(item, promise) {
      if (!item) return promise;
      const id = key(item), set = pending.get(id) || new Set(), sequence = ++serial;
      pending.set(id, set);
      set.add(promise);
      const remove = () => {
        set.delete(promise);
        if (!set.size && pending.get(id) === set) pending.delete(id);
      };
      Promise.resolve(promise).then(saved => {
        if (saved === true && failures.get(id)?.sequence < sequence) failures.delete(id);
        remove();
      }, error => {
        if (!failures.has(id) || failures.get(id).sequence < sequence) failures.set(id, { sequence, error });
        remove();
      });
      return promise;
    },
    cancel,
    async open(item) {
      cancel();
      const current = new AbortController();
      controller = current;
      const { signal } = current, id = key(item), deadline = Date.now() + timeoutMs;
      try {
        while (pending.has(id) && !signal.aborted) await wait(Promise.allSettled([...pending.get(id)]), signal, deadline);
        if (signal.aborted) return null;
        if (failures.has(id)) throw new Error('Previous video progress could not be saved. Reload to use the server checkpoint.');
        let progress = item.progress;
        try {
          const result = await wait(requestJSON(`/api/video-progress/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}`, { cache: 'no-store', signal }), signal, deadline);
          if (signal.aborted) return null;
          progress = result.data;
        } catch (error) {
          if (signal.aborted) return null;
          if (error.message.includes('timed out')) throw error;
          // Offline: retain the catalogue snapshot, protected by revision checks.
        }
        return signal.aborted ? null : { ...item, progress };
      } finally {
        if (controller === current) controller = null;
        current.abort();
      }
    },
  };
}

export function createVideoPlayer({ document, $, escapeHTML, requestJSON }) {
  mountVideoPlayer(document);
  const player = $("#video-player");
  const video = $("#video-element");
  const preview = $("#video-preview");
  const previewImage = $("#video-preview-image");
  let previewPosition = -1;
  let previewRequest = 0;
  const audioSelect = $("#audio-track");
  const subtitleSelect = $("#subtitle-track");
  let hlsPlayer;
  let hlsGeneration = 0;
  let activeItem;
  let activeHLSSource;
  const seekOrigin = createMediaSeekOrigin();

  function hidePreview() {
    preview.hidden = true;
    previewPosition = -1;
    previewRequest++;
  }

  function showPreview(event) {
    if (!activeItem || event.pointerType === "touch") return;
    const duration = Number(activeItem.duration_seconds || video.duration);
    if (!Number.isFinite(duration) || duration <= 0) return;
    const bounds = $("#video-seek").getBoundingClientRect();
    const fraction = Math.max(
      0,
      Math.min(1, (event.clientX - bounds.left) / bounds.width),
    );
    const seconds = Math.floor((fraction * duration) / 10) * 10;
    preview.style.setProperty("--preview-x", `${fraction * 100}%`);
    $("#video-preview-time").textContent = timeLabel(seconds) || "0:00";
    if (seconds === previewPosition) {
      preview.hidden = false;
      return;
    }
    previewPosition = seconds;
    const requestID = ++previewRequest;
    preview.hidden = true;
    previewImage.style.backgroundImage = "";
    const image = new Image();
    image.onload = () => {
      if (requestID !== previewRequest) return;
      previewImage.style.backgroundImage = `url(${JSON.stringify(image.src)})`;
      preview.hidden = false;
    };
    image.onerror = () => {
      if (requestID === previewRequest) hidePreview();
    };
    image.src = `/api/video-previews/${encodeURIComponent(activeItem.library_id)}/${encodeURIComponent(activeItem.id)}/${seconds}.jpg?v=${encodeURIComponent(activeItem.revision || "")}`;
  }

  $("#video-seek").addEventListener("pointermove", showPreview);
  $("#video-seek").addEventListener("pointerleave", hidePreview);
  createVideoControls({
    document, $, player, video,
    getDuration: () => Number(activeItem?.duration_seconds) || video.duration,
  });

  function updateRuntime(item, prepared = 0) {
    const total = Number(item?.duration_seconds);
    const totalText = timeLabel(total);
    $("#video-runtime").textContent =
      prepared > 0 && prepared < total - 6
        ? `${totalText} total · ${timeLabel(prepared)} ready`
        : totalText;
  }

  function removeSubtitleTracks() {
    video
      .querySelectorAll("track[data-bobobooks]")
      .forEach((track) => track.remove());
  }

  function setSubtitle(value) {
    const selected = String(value || "");
    subtitleSelect.value = selected;
    video.querySelectorAll("track[data-bobobooks]").forEach((track) => {
      track.track.mode =
        track.dataset.stream === selected ? "showing" : "disabled";
    });
  }

  function installSubtitleTracks(item, subtitles) {
    removeSubtitleTracks();
    subtitles.forEach((trackInfo) => {
      const track = document.createElement("track");
      track.dataset.bobobooks = "true";
      track.dataset.stream = String(trackInfo.index);
      track.kind = "subtitles";
      track.label = trackLabel(trackInfo, `Subtitle ${trackInfo.index}`);
      if (trackInfo.language) track.srclang = trackInfo.language;
      track.src = `/api/video-subtitles/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}/${encodeURIComponent(trackInfo.index)}.vtt?v=${encodeURIComponent(item.revision || "")}`;
      track.onload = () => setSubtitle(subtitleSelect.value);
      track.onerror = () => {
        $("#video-message").textContent =
          "These subtitles could not be prepared.";
      };
      video.append(track);
    });
  }

  function prepareTrackMenu(item) {
    const audio = item.audio_streams || [];
    const subtitles = (item.subtitle_streams || []).filter(
      (track) => track.text,
    );
    $("#track-menu").hidden = audio.length < 2 && subtitles.length === 0;
    $("#track-menu").open = false;
    $("#audio-track-field").hidden = audio.length < 2;
    $("#subtitle-track-field").hidden = subtitles.length === 0;
    audioSelect.innerHTML = audio
      .map(
        (track, order) =>
          `<option value="${track.index}">${escapeHTML(trackLabel(track, `Audio ${order + 1}`))}</option>`,
      )
      .join("");
    audioSelect.value = String(
      (
        preferredVideoTrack(audio, item.playback_preferences?.audio_language) ||
        audio.find((track) => track.default) ||
        audio[0]
      )?.index ?? "",
    );
    audioSelect.disabled = audio.length < 2 || !item.hls_url;
    subtitleSelect.innerHTML = `<option value="">Off</option>${subtitles.map((track) => `<option value="${track.index}">${escapeHTML(trackLabel(track, `Subtitle ${track.index}`))}${track.forced ? " · Forced" : ""}</option>`).join("")}`;
    installSubtitleTracks(item, subtitles);
    const subtitlePreference = item.playback_preferences?.subtitle_language;
    const subtitle =
      subtitlePreference && subtitlePreference !== "off"
        ? preferredVideoTrack(subtitles, subtitlePreference)
        : undefined;
    setSubtitle(String(subtitle?.index ?? ""));
  }

  function enableNativeAudioTracks() {
    const tracks = video.audioTracks;
    if (!tracks || tracks.length < 2) return;
    audioSelect.disabled = false;
    const selected = (activeItem?.audio_streams || []).findIndex(
      (track) => track.index === Number(audioSelect.value),
    );
    for (let index = 0; index < tracks.length; index++)
      tracks[index].enabled = index === selected;
  }

  function destroyHLS() {
    hlsGeneration++;
    hlsPlayer?.destroy();
    hlsPlayer = undefined;
    stopServerTranscode();
  }

  // Tells the server to stop encoding the HLS stream it had been preparing,
  // instead of leaving that up to the passive ~75s idle timeout. Best
  // effort: a lost connection here just means the idle timeout handles it
  // instead, no worse than before this existed.
  function stopServerTranscode() {
    if (!activeHLSSource) return;
    const url = activeHLSSource;
    activeHLSSource = undefined;
    fetch(url, { method: "DELETE", keepalive: true }).catch(() => {});
  }

  async function attachHLS(item, audioIndex, resumeAt = 0, quality = effectivePlaybackQuality()) {
    $("#video-playback-mode").textContent = ` · Transcoding · ${quality[0].toUpperCase()}${quality.slice(1)}`;
    destroyHLS();
    const generation = hlsGeneration;
    video.removeAttribute("src");
    video.load();
    const source = videoHLSURL(item, audioIndex, quality === 'maximum' ? 'high' : quality);
    activeHLSSource = source;
    const knownDuration = Number(item.duration_seconds);
    const resumePosition = videoResumePosition(knownDuration, resumeAt);
    if (resumePosition) seekOrigin.expectAutomatic(resumePosition);
    const begin = () =>
      video.play().catch(() => {
        $("#video-message").textContent = "Press play to start this video.";
      });
    let Hls;
    try {
      Hls = await loadHLSClient();
    } catch (error) {
      if (generation !== hlsGeneration) return false;
      if (!video.canPlayType("application/vnd.apple.mpegurl")) throw error;
    }
    if (generation !== hlsGeneration) return false;
    if (Hls?.isSupported()) {
      hlsPlayer = new Hls({
        maxBufferLength: 30,
        backBufferLength: 30,
        maxBufferSize: 60 * 1024 * 1024,
        startPosition: resumePosition || -1,
      });
      hlsPlayer.attachMedia(video);
      hlsPlayer.on(Hls.Events.MEDIA_ATTACHED, () =>
        hlsPlayer?.loadSource(source),
      );
      hlsPlayer.on(Hls.Events.MANIFEST_PARSED, begin);
      hlsPlayer.on(Hls.Events.LEVEL_UPDATED, (_event, data) =>
        updateRuntime(item, data.details?.edge || 0),
      );
      // A hiccup (a slow segment, a decoder stumble) is retried before giving up.
      let recoveries = 0;
      hlsPlayer.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal) return;
        console.error("Compatibility playback error", data.type, data.details, data.response?.code);
        if (recoveries < 3 && data.type === Hls.ErrorTypes.NETWORK_ERROR) {
          recoveries += 1;
          $("#video-message").textContent = "Reconnecting…";
          setTimeout(() => hlsPlayer?.startLoad(), 1500 * recoveries);
          return;
        }
        if (recoveries < 3 && data.type === Hls.ErrorTypes.MEDIA_ERROR) {
          recoveries += 1;
          if (recoveries === 2) hlsPlayer.swapAudioCodec();
          hlsPlayer.recoverMediaError();
          return;
        }
        $("#video-message").textContent =
          `Compatibility playback stopped (${data.details || data.type}). Close the player and try again.`;
      });
      hlsPlayer.on(Hls.Events.FRAG_LOADED, () => {
        if ($("#video-message").textContent === "Reconnecting…") $("#video-message").textContent = "";
      });
      return true;
    }
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = source;
      video.onloadedmetadata = () => {
        if (resumePosition && resumePosition < video.duration - 5) {
          seekOrigin.expectAutomatic(resumePosition);
          video.currentTime = resumePosition;
        }
        begin();
      };
      return true;
    }
    throw new Error("Compatibility playback is not supported by this browser");
  }

  audioSelect.onchange = () => {
    if (!activeItem) return;
    const tracks = video.audioTracks;
    if (tracks?.length >= 2 && !hlsPlayer) {
      enableNativeAudioTracks();
      return;
    }
    const position = Number.isFinite(video.currentTime) ? video.currentTime : 0;
    $("#video-message").textContent = "Switching audio…";
    attachHLS(activeItem, Number(audioSelect.value), position)
      .then((attached) => {
        if (attached) $("#video-message").textContent = "";
      })
      .catch((error) => {
        $("#video-message").textContent = error.message;
      });
  };
  subtitleSelect.onchange = () => {
    setSubtitle(subtitleSelect.value);
    $("#track-menu").open = false;
  };
  const deviceID =
    localStorage.getItem("bobobooks-video-device") ||
    crypto.randomUUID?.() ||
    `${Date.now()}-${Math.random()}`;
  localStorage.setItem("bobobooks-video-device", deviceID);
  const progress = createVideoProgress({ requestJSON, getActive: () => activeItem, getVideo: () => video, deviceID });
  const openProgress = createVideoOpenProgress(requestJSON);
  function saveProgress(force = false) {
    return openProgress.track(activeItem, progress.save(force));
  }
  function closePlayer() {
    openProgress.cancel();
    hidePreview();
    saveProgress(true);
    // Relinquish item ownership before pause dispatches. The pause listener
    // must not enqueue a duplicate final checkpoint for a player the user has
    // already closed, and later callbacks must not retain a stale item.
    activeItem = null;
    video.pause();
    destroyHLS();
    removeSubtitleTracks();
    video.removeAttribute("src");
    video.load();
    seekOrigin.clear();
    if (player.open) player.close();
  }
  async function openPlayer(item, title) {
    if (!item) return;
    if (activeItem) closePlayer();
    try {
      item = await openProgress.open(item);
    } catch (error) {
      window.alert(error.message);
      return;
    }
    if (!item) return;
    activeItem = item;
    seekOrigin.clear();
    progress.reset(item);
    $("#video-title").textContent = title;
    updateRuntime(item);
    $("#video-message").textContent = "";
    try {
      const query = new URLSearchParams({
        library: item.library_id,
        show: item.show || "",
      });
      item.playback_preferences = (
        await requestJSON(`/api/video-preferences?${query}`)
      ).data.effective;
    } catch (_) {
      item.playback_preferences = {};
    }
    if (activeItem !== item) return;
    prepareTrackMenu(item);
    player.showModal();
    const beginDirect = () => {
      const position = videoResumePosition(video.duration, item.progress?.position_seconds, item.progress?.finished);
      if (position) {
        seekOrigin.expectAutomatic(position);
        video.currentTime = position;
      }
      video.play().catch(() => {
        $("#video-message").textContent = "Press play to start this video.";
      });
    };
    const quality = effectivePlaybackQuality();
    if (item.stream_url && quality === 'maximum') {
      $("#video-playback-mode").textContent = " · Direct play";
      video.src = item.stream_url;
      video.onloadedmetadata = () => {
        enableNativeAudioTracks();
        beginDirect();
      };
    } else {
      $("#video-playback-mode").textContent = " · Transcoding";
      const audio =
        (item.audio_streams || []).find(
          (track) => String(track.index) === audioSelect.value,
        ) || item.audio_streams?.[0];
      $("#video-message").textContent = "Preparing compatibility playback…";
      attachHLS(
        item,
        audio?.index,
        videoResumePosition(item.duration_seconds, item.progress?.position_seconds, item.progress?.finished),
        quality,
      )
        .then((attached) => {
          if (attached) $("#video-message").textContent = "";
        })
        .catch((error) => {
          $("#video-message").textContent = error.message;
        });
    }
  }
  $("#video-close").onclick = closePlayer;
  player.addEventListener("cancel", (event) => {
    event.preventDefault();
    closePlayer();
  });
  player.addEventListener("click", (event) => {
    if (event.target === player) closePlayer();
  });
  video.addEventListener("timeupdate", () => saveProgress(false));
  video.addEventListener("seeking", () => {
    if (seekOrigin.isExplicit(video.currentTime)) progress.claim();
  });
  video.addEventListener("play", () => {
    progress.activate();
  });
  video.addEventListener("playing", () => seekOrigin.clear());
  video.addEventListener("pause", () => saveProgress(true));
  video.addEventListener("ended", () => saveProgress(true));
  const lockLandscape = () =>
    screen.orientation?.lock?.("landscape").catch(() => {});
  const unlockOrientation = () => screen.orientation?.unlock?.();
  document.addEventListener("fullscreenchange", () =>
    document.fullscreenElement ? lockLandscape() : unlockOrientation(),
  );
  video.addEventListener("webkitbeginfullscreen", lockLandscape);
  video.addEventListener("webkitendfullscreen", unlockOrientation);
  // Closing the tab or navigating away never runs the close button's click
  // handler. Without this, the server would keep transcoding until its own
  // ~75s idle timeout noticed; sendBeacon survives the page unloading where
  // a plain fetch would not.
  window.addEventListener("pagehide", () => {
    if (activeHLSSource) navigator.sendBeacon?.(activeHLSSource);
  });

  return { open: openPlayer, abort: () => {
    openProgress.cancel();
    if (activeItem && !player.open) closePlayer();
  } };
}
