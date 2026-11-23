// lib/hls.ts — loads the HLS client that this server already ships at /assets/vendor/hls.js/hls.light.min.js.
//
// No npm dependency on purpose. The legacy player (static/hls-client.js) has been loading that exact vendor file for
// a while, so it is already in the image, already cached by the browser, and already the version the transcoder's
// playlists are tested against. Pulling hls.js into the bundle instead would add ~110 kB gzip to a bundle with a
// 170 kB budget for a feature that only video pages need; loaded this way it costs the browsing app nothing at all.
//
// Typed by hand, and only with what is actually called here: hls.js's own types are large, and a `any`-shaped global
// would hide a typo in an event name that only fails when a film is opened.
export interface HlsInstance {
  attachMedia(media: HTMLMediaElement): void;
  loadSource(source: string): void;
  on(event: string, listener: (event: string, data: HlsEventData) => void): void;
  destroy(): void;
}

export interface HlsEventData {
  fatal?: boolean;
  type?: string;
  details?: string;
  level?: number;
  startPosition?: number;
}

interface HlsConstructor {
  new (config: Record<string, unknown>): HlsInstance;
  isSupported(): boolean;
  Events: { MEDIA_ATTACHED: string; MANIFEST_PARSED: string; ERROR: string };
}

declare global {
  interface Window {
    Hls?: HlsConstructor;
  }
}

const HLS_SOURCE = "/assets/vendor/hls.js/hls.light.min.js?v=1";

let loader: Promise<HlsConstructor> | undefined;

/** Resolves once window.Hls exists. One script tag per page, shared by every player instance. */
export function loadHls(): Promise<HlsConstructor> {
  if (window.Hls !== undefined) return Promise.resolve(window.Hls);
  if (loader !== undefined) return loader;
  loader = new Promise<HlsConstructor>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = HLS_SOURCE;
    script.async = true;
    script.onload = () => {
      if (window.Hls !== undefined) resolve(window.Hls);
      else reject(new Error("The HLS client did not initialise."));
    };
    script.onerror = () => reject(new Error("The HLS client could not be loaded."));
    document.head.append(script);
  });
  // A failed load must be retryable: clear the cached promise so the next attempt makes a fresh request rather than
  // replaying the same rejection forever in a tab that stays open.
  loader.catch(() => {
    loader = undefined;
  });
  return loader;
}

/** True when this browser plays HLS by itself — Safari and iOS do, which is why hls.js is a fallback and not a
 *  requirement. */
export function nativeHlsSupport(video: HTMLVideoElement): boolean {
  return video.canPlayType("application/vnd.apple.mpegurl") !== "";
}
