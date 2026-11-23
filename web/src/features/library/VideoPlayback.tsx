// features/library/VideoPlayback.tsx — the full-screen player itself: the video element, the chrome over it, HLS
// attach, audio/subtitle switching and progress. A feature component rather than a screen, because it owns layout
// (law L2: screens compose primitives, features may own CSS).
//
// The chrome imitates no mock-up, because there is no player sheet — Kevin's reference was Plex ("i prefer a bit more
// plex styled video playback compared to seeing embedded video on the web browser like that"), which is what this is:
// the film owns the screen, the controls fade away while watching, and they come back on any movement or tap.
//
// Behaviour worth knowing:
//   * controls hide after a few seconds **while playing** and never while paused; any pointer movement, tap or key
//     wakes them (a paused player that hides its controls strands the reader);
//   * the seek bar is a real <input type=range> drawn behind a painted track, so dragging, keyboard and screen readers
//     all work without a custom gesture;
//   * `timeupdate` is ignored while the reader is dragging, or the thumb fights the playhead;
//   * keyboard: space/k play-pause, ←/→ ±15 s, f fullscreen, Escape back;
//   * an audio track is a different playlist URL (the server's stream carries the audio index), so switching re-attaches
//     at the current playhead;
//   * subtitles are WebVTT <track> elements, and only tracks the probe marked `text` are offered — image-based PGS
//     cannot be converted, so listing them would be offering a control that always fails.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AudioLines, Subtitles } from "lucide-react";
import { Menu, Text } from "../../ui";
// Imported directly, not through ui/index: the barrel is shared by every screen, so re-exporting the player's
// primitives there would pull Radix Dialog and the player chrome into the entry chunk and undo the route split.
import { CaptionOverlay } from "../../ui/CaptionOverlay";
import { PlaybackSettingsSheet } from "../../ui/PlaybackSettingsSheet";
import { VideoPlayerSurface } from "../../ui/VideoPlayerSurface";
import { useActiveCues } from "./useActiveCues";
import { loadHls, nativeHlsSupport, type HlsInstance } from "../../lib/hls";
import {
  cueStyle,
  loadSubtitlePreferences,
  saveSubtitlePreferences,
  type SubtitlePreferences,
} from "../../lib/subtitlePreferences";
import { saveVideoProgress, saveVideoSeek } from "./api/videoProgress";
import type { MediaTrack } from "./api/mediaDetail";
import { ImageCaptions } from "./ImageCaptions";

const PROGRESS_TICK_MS = 15_000;
const CONTROLS_IDLE_MS = 3_000;
const SEEK_STEP_S = 15;
const RESUME_THRESHOLD_S = 5;
const RESUME_END_GUARD_S = 10;

function hlsUrlFor(libraryId: string, itemId: string, audioIndex: number): string {
  return `/api/video-hls/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}/${audioIndex}/index.m3u8`;
}

function trackLabel(track: MediaTrack, fallback: string): string {
  if (track.title !== undefined && track.title !== "") return track.title;
  if (track.language !== undefined && track.language !== "") return track.language.toUpperCase();
  return fallback;
}

export interface VideoPlaybackProps {
  libraryId: string;
  itemId: string;
  title: string;
  mediaRevision: string;
  streamUrl: string | undefined;
  hlsUrl: string | undefined;
  positionSeconds: number;
  durationSeconds: number;
  audioTracks: MediaTrack[];
  subtitleTracks: MediaTrack[];
  onBack: () => void;
  onOpenDetail: () => void;
}

export function VideoPlayback({
  libraryId,
  itemId,
  title,
  mediaRevision,
  streamUrl,
  hlsUrl,
  positionSeconds,
  durationSeconds,
  audioTracks,
  subtitleTracks,
  onBack,
  onOpenDetail,
}: VideoPlaybackProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const shellRef = useRef<HTMLDivElement>(null);

  const [playing, setPlaying] = useState(false);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | undefined>(undefined);
  const [position, setPosition] = useState(positionSeconds);
  const [duration, setDuration] = useState(durationSeconds);
  const [buffered, setBuffered] = useState(0);
  const [controlsVisible, setControlsVisible] = useState(true);
  const [fullscreen, setFullscreen] = useState(false);
  /** While the reader drags the bar, `timeupdate` must not fight the thumb. */
  const [scrubbing, setScrubbing] = useState(false);

  /** Subtitle appearance and speed are per-device preferences (lib/subtitlePreferences), read once and applied live. */
  const [preferences, setPreferences] = useState<SubtitlePreferences>(() => loadSubtitlePreferences());
  const [settingsOpen, setSettingsOpen] = useState(false);

  const defaultAudio = audioTracks.find((track) => track.default === true) ?? audioTracks[0];
  const [audioIndex, setAudioIndex] = useState<number | undefined>(defaultAudio?.index);
  const [subtitleIndex, setSubtitleIndex] = useState<number | undefined>(undefined);
  // Where to land after a track switch. Held as an object so the play state survives too — changing the audio track
  // must not silently pause a film the person was watching.
  //
  // It is read and cleared by the *source* effect body, never by that effect's cleanup. Cleanup runs first on every
  // dependency change, so clearing it there wiped the position before the new stream could read it and every track
  // switch restarted the film from the beginning, which is exactly what Kevin reported.
  const switchAt = useRef<{ at: number; playing: boolean } | undefined>(undefined);
  const reported = useRef(positionSeconds);

  const openAt =
    positionSeconds > RESUME_THRESHOLD_S && (durationSeconds === 0 || positionSeconds < durationSeconds - RESUME_END_GUARD_S)
      ? positionSeconds
      : 0;

  const playlist = useMemo(() => {
    if (streamUrl !== undefined) return undefined;
    if (audioIndex !== undefined) return hlsUrlFor(libraryId, itemId, audioIndex);
    return hlsUrl;
  }, [streamUrl, hlsUrl, libraryId, itemId, audioIndex]);

  // ---- source -------------------------------------------------------------------------------------------------
  useEffect(() => {
    const video = videoRef.current;
    if (video === null || playlist === undefined) return;
    if (streamUrl !== undefined) {
      video.src = streamUrl;
      return;
    }

    // Consumed once, here: a re-run for any other reason (a progress prop, a remount) must fall back to openAt
    // rather than seek back to a stale switch position.
    const carry = switchAt.current;
    switchAt.current = undefined;
    const startAt = carry?.at ?? openAt;
    let hls: HlsInstance | undefined;
    let cancelled = false;

    async function attach() {
      const element = videoRef.current;
      if (element === null) return;
      // hls.js **first**, native second — the opposite order looks safer and is wrong: Chromium answers "maybe" to
      // canPlayType("application/vnd.apple.mpegurl") and then cannot decode the playlist at all.
      try {
        const Hls = await loadHls();
        if (cancelled || videoRef.current === null) return;
        if (Hls.isSupported()) {
          hls = new Hls({
            maxBufferLength: 30,
            backBufferLength: 30,
            maxBufferSize: 60 * 1024 * 1024,
            startPosition: startAt > 0 ? startAt : -1,
          });
          hls.attachMedia(videoRef.current);
          hls.on(Hls.Events.MEDIA_ATTACHED, () => hls?.loadSource(playlist!));
          hls.on(Hls.Events.MANIFEST_PARSED, () => {
            if (videoRef.current === null) return;
            videoRef.current.currentTime = startAt;
            // Only a track switch resumes by itself; an ordinary open still waits for the person to press play.
            if (carry?.playing === true) void videoRef.current.play().catch(() => undefined);
          });
          hls.on(Hls.Events.ERROR, (_event, data) => {
            if (data.fatal !== true) return;
            if (data.type === "networkError") {
              hls?.loadSource(playlist!);
              return;
            }
            setError("Playback stopped. Try again.");
          });
          return;
        }
      } catch {
        if (cancelled) return;
      }
      if (nativeHlsSupport(element)) {
        element.src = playlist!;
        element.currentTime = startAt;
        return;
      }
      setError("This browser cannot play transcoded video.");
    }

    void attach();
    return () => {
      cancelled = true;
      hls?.destroy();
    };
  }, [playlist, streamUrl, openAt]);

  // ---- media events drive the chrome --------------------------------------------------------------------------
  useEffect(() => {
    const video = videoRef.current;
    if (video === null) return;
    const onTime = () => {
      if (!scrubbing) setPosition(video.currentTime);
      if (Number.isFinite(video.duration)) setDuration(video.duration);
      const ranges = video.buffered;
      if (ranges.length > 0) setBuffered(ranges.end(ranges.length - 1));
    };
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onCanPlay = () => setReady(true);
    const onMeta = () => {
      if (Number.isFinite(video.duration)) setDuration(video.duration);
    };
    video.addEventListener("timeupdate", onTime);
    video.addEventListener("progress", onTime);
    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("canplay", onCanPlay);
    video.addEventListener("loadedmetadata", onMeta);
    return () => {
      video.removeEventListener("timeupdate", onTime);
      video.removeEventListener("progress", onTime);
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("canplay", onCanPlay);
      video.removeEventListener("loadedmetadata", onMeta);
    };
  }, [scrubbing]);

  // ---- progress -----------------------------------------------------------------------------------------------
  useEffect(() => {
    const video = videoRef.current;
    if (video === null) return;

    function write(force: boolean) {
      const element = videoRef.current;
      if (element === null || element.currentTime === 0 || !Number.isFinite(element.duration)) return;
      if (!force && Math.abs(element.currentTime - reported.current) < 1) return;
      reported.current = element.currentTime;
      const payload = {
        libraryId,
        itemId,
        positionSeconds: element.currentTime,
        durationSeconds: element.duration,
        mediaRevision,
      };
      void saveVideoProgress(payload).catch(async () => {
        // A checkpoint that moves backwards is a deliberate seek, which the server refuses by design; retrying as an
        // explicit seek is what makes rewinding stick.
        try {
          await saveVideoSeek(payload);
        } catch {
          // Progress is a convenience; a failed save must not interrupt the film.
        }
      });
    }

    const onPause = () => write(true);
    const onEnded = () => write(true);
    const tick = window.setInterval(() => {
      if (!video.paused) write(false);
    }, PROGRESS_TICK_MS);
    const onHidden = () => {
      if (document.visibilityState === "hidden") write(true);
    };

    video.addEventListener("pause", onPause);
    video.addEventListener("ended", onEnded);
    document.addEventListener("visibilitychange", onHidden);
    window.addEventListener("pagehide", onHidden);
    return () => {
      window.clearInterval(tick);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("ended", onEnded);
      document.removeEventListener("visibilitychange", onHidden);
      window.removeEventListener("pagehide", onHidden);
      write(true);
    };
  }, [libraryId, itemId, mediaRevision]);

  // ---- preferences --------------------------------------------------------------------------------------------
  // Speed applies to the element directly (it survives a source switch because it is element state, not playlist
  // state), and everything is written back so the next film opens with the same look and pace.
  useEffect(() => {
    const video = videoRef.current;
    if (video !== null) video.playbackRate = preferences.speed;
    saveSubtitlePreferences(preferences);
  }, [preferences, ready]);

  // ---- subtitles ----------------------------------------------------------------------------------------------
  useEffect(() => {
    const video = videoRef.current;
    if (video === null) return;
    const tracks = video.textTracks;
    for (let i = 0; i < tracks.length; i += 1) {
      const track = tracks[i];
      if (track === undefined) continue;
      const index = track.id === "" ? Number.NaN : Number(track.id);
      track.mode = index === subtitleIndex ? "showing" : "disabled";
    }
  }, [subtitleIndex, subtitleTracks, ready]);

  // ---- chrome auto-hide ---------------------------------------------------------------------------------------
  const idleTimer = useRef<number | undefined>(undefined);
  const wake = useCallback(() => {
    setControlsVisible(true);
    window.clearTimeout(idleTimer.current);
    idleTimer.current = window.setTimeout(() => {
      const video = videoRef.current;
      if (video !== null && !video.paused) setControlsVisible(false);
    }, CONTROLS_IDLE_MS);
  }, []);

  // Arm the timer once on mount. The setState lives inside the timeout callback, not in the effect body.
  useEffect(() => {
    const id = window.setTimeout(() => {
      const video = videoRef.current;
      if (video !== null && !video.paused) setControlsVisible(false);
    }, CONTROLS_IDLE_MS);
    return () => window.clearTimeout(id);
  }, []);

  // Paused players keep their controls (the idle timer only hides them while playing, above), so no effect is needed:
  // `wake` decides, and it is called from every interaction.

  // A phone held upright shows a 16:9 film in a letterbox two-thirds the height of the screen. Plex rotates; so does
  // this, where the platform allows it — only inside the gesture that started playback, and only where the API exists
  // (Android Chrome), silently doing nothing everywhere else (iOS Safari has no orientation lock).
  const lockedOrientation = useRef(false);
  useEffect(() => {
    const orientation = screen.orientation as ScreenOrientation & { lock?: (mode: string) => Promise<void> };
    if (typeof orientation?.lock !== "function") return;
    const unlock = () => {
      if (!lockedOrientation.current) return;
      lockedOrientation.current = false;
      try {
        orientation.unlock();
      } catch {
        // Nothing to do: the browser will restore its own orientation on the next rotation.
      }
    };
    window.addEventListener("pointerup", () => {
      if (lockedOrientation.current) return;
      orientation.lock?.("landscape").then(
        () => {
          lockedOrientation.current = true;
        },
        () => {
          // Refused (not fullscreen, or unsupported): portrait playback stays letterboxed, which is fine.
        },
      );
    }, { once: true });
    return () => {
      unlock();
    };
  }, []);

  function togglePlay() {
    const video = videoRef.current;
    if (video === null) return;
    if (video.paused) void video.play().catch(() => setError("This browser blocked playback. Tap play again."));
    else video.pause();
    wake();
  }

  function seekBy(delta: number) {
    const video = videoRef.current;
    if (video === null) return;
    const limit = Number.isFinite(video.duration) ? video.duration : video.currentTime + delta;
    video.currentTime = Math.min(Math.max(0, video.currentTime + delta), limit);
    setPosition(video.currentTime);
    wake();
  }

  function seekTo(seconds: number) {
    const video = videoRef.current;
    if (video === null) return;
    video.currentTime = seconds;
    setPosition(seconds);
  }

  const toggleFullscreen = useCallback(async () => {
    try {
      if (document.fullscreenElement !== null) await document.exitFullscreen();
      else await shellRef.current?.requestFullscreen();
    } catch {
      // A browser that refuses fullscreen simply stays windowed; playback is unaffected.
    }
  }, []);

  useEffect(() => {
    const onChange = () => setFullscreen(document.fullscreenElement !== null);
    document.addEventListener("fullscreenchange", onChange);
    return () => document.removeEventListener("fullscreenchange", onChange);
  }, []);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      const video = videoRef.current;
      if (video === null) return;
      // The settings sheet owns the keyboard while it is open: Escape must close the panel, not walk out of the film,
      // and space must not pause the video while a button in the panel has focus.
      if (settingsOpen) return;
      if (event.key === " " || event.key === "k") {
        event.preventDefault();
        togglePlay();
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        seekBy(SEEK_STEP_S);
      } else if (event.key === "ArrowLeft") {
        event.preventDefault();
        seekBy(-SEEK_STEP_S);
      } else if (event.key === "f") {
        void toggleFullscreen();
      } else if (event.key === "Escape" && document.fullscreenElement === null) {
        onBack();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  function switchAudio(index: number) {
    if (index === audioIndex) return;
    const video = videoRef.current;
    const at = video !== null && video.currentTime > 0 ? video.currentTime : openAt;
    switchAt.current = { at, playing: video !== null && !video.paused };
    setReady(false);
    setError(undefined);
    setAudioIndex(index);
  }

  // Text tracks become <track> elements and are drawn by CaptionOverlay. Image tracks (PGS — what a 1080p remux
  // actually ships) cannot be WebVTT at all: the server renders them to cue PNGs and ImageCaptions draws those.
  const chosenSubtitle = subtitleTracks.find((track) => track.index === subtitleIndex);
  const imageSubtitleIndex = chosenSubtitle !== undefined && chosenSubtitle.text !== true ? chosenSubtitle.index : undefined;
  const playableSubtitles = subtitleTracks.filter((track) => track.text === true);
  const captionLines = useActiveCues(subtitleIndex);
  const bufferedPercent = duration > 0 ? (buffered / duration) * 100 : 0;

  if (error !== undefined) {
    return (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: "var(--space-12)",
          width: "100%",
          height: "100%",
          background: "var(--color-bg)",
        }}
      >
        <Text size="14" tone="error">{error}</Text>
        <button
          type="button"
          onClick={onOpenDetail}
          style={{
            padding: "var(--space-8) var(--space-16)",
            border: "none",
            borderRadius: "var(--radius-pill)",
            background: "var(--color-raised)",
            color: "var(--color-text)",
            fontSize: "var(--font-size-13)",
            cursor: "pointer",
          }}
        >
          Back to the film
        </button>
      </div>
    );
  }

  return (
    <VideoPlayerSurface
      title={title}
      ready={ready}
      playing={playing}
      controlsVisible={controlsVisible}
      fullscreen={fullscreen}
      position={position}
      duration={duration}
      bufferedPercent={bufferedPercent}
      onBack={onBack}
      onTogglePlay={togglePlay}
      onSeekBy={seekBy}
      onSeekTo={seekTo}
      onSeekStart={() => setScrubbing(true)}
      onSeekEnd={() => setScrubbing(false)}
      onToggleFullscreen={() => void toggleFullscreen()}
      onOpenSettings={() => {
        setSettingsOpen(true);
        wake();
      }}
      onWake={wake}
      cueVars={cueStyle(preferences)}
      captions={
        <>
          <CaptionOverlay lines={captionLines} lifted={controlsVisible} style={cueStyle(preferences)} />
          {imageSubtitleIndex !== undefined && (
            <ImageCaptions
              libraryId={libraryId as string}
              itemId={itemId}
              streamIndex={imageSubtitleIndex}
              position={position}
            />
          )}
        </>
      }
      menus={
        <>
          {audioTracks.length > 1 && (
            <Menu
              label="Audio track"
              icon={AudioLines}
              items={audioTracks.map((track) => ({ id: String(track.index), label: trackLabel(track, `Track ${track.index}`) }))}
              activeIds={audioIndex === undefined ? [] : [String(audioIndex)]}
              onSelect={(id) => switchAudio(Number(id))}
            />
          )}
          {subtitleTracks.length > 0 && (
            <Menu
              label="Subtitles"
              icon={Subtitles}
              items={[
                { id: "off", label: "Off" },
                ...subtitleTracks.map((track) => ({
                  id: String(track.index),
                  // Say which tracks have to be prepared: an image track is selected instantly but its first caption
                  // appears when the server has finished decoding it.
                  label: track.text === true ? trackLabel(track, `Track ${track.index}`) : `${trackLabel(track, `Track ${track.index}`)} (image)`,
                })),
              ]}
              activeIds={[subtitleIndex === undefined ? "off" : String(subtitleIndex)]}
              onSelect={(id) => setSubtitleIndex(id === "off" ? undefined : Number(id))}
            />
          )}
        </>
      }
      video={
        <video
          ref={videoRef}
          playsInline
          preload="metadata"
          style={{ display: "block", width: "100%", height: "100%", objectFit: "contain", background: "#000" }}
          onClick={() => {
            if (controlsVisible) togglePlay();
            else wake();
          }}
          onDoubleClick={() => seekBy(SEEK_STEP_S)}
        >
          {playableSubtitles.map((track) => (
            <track
              key={track.index}
              id={String(track.index)}
              kind="subtitles"
              src={`/api/video-subtitles/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}/${track.index}.vtt`}
              srcLang={track.language === undefined || track.language === "" ? "und" : track.language}
              label={trackLabel(track, `Track ${track.index}`)}
            />
          ))}
        </video>
      }
      overlay={
        <PlaybackSettingsSheet
          open={settingsOpen}
          onOpenChange={setSettingsOpen}
          preferences={preferences}
          onChange={(patch) => setPreferences((previous) => ({ ...previous, ...patch }))}
          hasSubtitles={playableSubtitles.length > 0}
        />
      }
    />
  );
}
