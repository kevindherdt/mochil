// lib/subtitlePreferences.ts — how subtitles look and how fast films play, remembered per device.
//
// Deliberately *not* in /api/settings: that document is the server's own configuration, it is administrator-only, and
// these are personal viewing preferences — a guest watching on a phone should be able to make the captions bigger
// without touching the server's config, and two people sharing a server should not overwrite each other's font size.
// localStorage is what the theme and the last-used shelf already use, so this needs no new machinery.
//
// The values map onto `::cue`, the one styling surface browsers expose for WebVTT. That surface is deliberately small:
// colour, background, text-shadow and font-size are the parts that work; anything about *where* a cue sits is decided
// by the cue itself or by the user agent, so there is no "move the subtitles up" setting here and none is pretended.
export type SubtitleSize = "small" | "medium" | "large" | "huge";
export type SubtitleColour = "white" | "yellow" | "black";
export type SubtitleBackground = "none" | "soft" | "solid";
export type SubtitleEdge = "outline" | "shadow" | "none";

export interface SubtitlePreferences {
  size: SubtitleSize;
  colour: SubtitleColour;
  background: SubtitleBackground;
  edge: SubtitleEdge;
  /** Playback rate, 0.5–2. Kept here because it is the other thing a viewer adjusts mid-film and expects remembered. */
  speed: number;
}

export const SUBTITLE_DEFAULTS: SubtitlePreferences = {
  size: "medium",
  colour: "white",
  background: "soft",
  edge: "outline",
  speed: 1,
};

export const SUBTITLE_SIZE_OPTIONS: { id: SubtitleSize; label: string }[] = [
  { id: "small", label: "Small" },
  { id: "medium", label: "Medium" },
  { id: "large", label: "Large" },
  { id: "huge", label: "Huge" },
];

export const SUBTITLE_COLOUR_OPTIONS: { id: SubtitleColour; label: string }[] = [
  { id: "white", label: "White" },
  { id: "yellow", label: "Yellow" },
  { id: "black", label: "Black" },
];

export const SUBTITLE_BACKGROUND_OPTIONS: { id: SubtitleBackground; label: string }[] = [
  { id: "none", label: "None" },
  { id: "soft", label: "Soft" },
  { id: "solid", label: "Solid" },
];

export const SUBTITLE_EDGE_OPTIONS: { id: SubtitleEdge; label: string }[] = [
  { id: "outline", label: "Outline" },
  { id: "shadow", label: "Shadow" },
  { id: "none", label: "None" },
];

export const SPEED_OPTIONS: { id: string; label: string }[] = [
  { id: "0.75", label: "0.75×" },
  { id: "1", label: "Normal" },
  { id: "1.25", label: "1.25×" },
  { id: "1.5", label: "1.5×" },
  { id: "2", label: "2×" },
];

const STORAGE_KEY = "mochil:subtitles";

/** Reads the stored preferences, falling back to the defaults for anything missing or unrecognised — a value written
 *  by a newer version must not make an older tab fail to render subtitles at all. */
export function loadSubtitlePreferences(): SubtitlePreferences {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (raw === null) return SUBTITLE_DEFAULTS;
    const stored = JSON.parse(raw) as Partial<SubtitlePreferences>;
    const pick = <T extends string>(value: unknown, allowed: readonly T[], fallback: T): T =>
      typeof value === "string" && (allowed as readonly string[]).includes(value) ? (value as T) : fallback;
    const speed = typeof stored.speed === "number" && stored.speed >= 0.5 && stored.speed <= 2 ? stored.speed : 1;
    return {
      size: pick(stored.size, SUBTITLE_SIZE_OPTIONS.map((option) => option.id), SUBTITLE_DEFAULTS.size),
      colour: pick(stored.colour, SUBTITLE_COLOUR_OPTIONS.map((option) => option.id), SUBTITLE_DEFAULTS.colour),
      background: pick(stored.background, SUBTITLE_BACKGROUND_OPTIONS.map((option) => option.id), SUBTITLE_DEFAULTS.background),
      edge: pick(stored.edge, SUBTITLE_EDGE_OPTIONS.map((option) => option.id), SUBTITLE_DEFAULTS.edge),
      speed,
    };
  } catch {
    return SUBTITLE_DEFAULTS;
  }
}

export function saveSubtitlePreferences(preferences: SubtitlePreferences): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
  } catch {
    // No storage (private window, full quota): the preferences simply do not persist. Never worth failing over.
  }
}

/** One table, two shapes: the `::cue` rule needs a length (`1.55em`, relative to whatever size the user agent gives a
 *  cue) and the sheet's preview needs a bare multiplier (`calc(16px * 1.55)`, because multiplying a length by an `em`
 *  is invalid CSS and silently falls back — which is exactly how the preview first shipped, stuck at one size). */
const CUE_SCALE: Record<SubtitleSize, number> = {
  small: 0.85,
  medium: 1,
  large: 1.25,
  huge: 1.55,
};

const CUE_COLOUR: Record<SubtitleColour, string> = {
  white: "#ffffff",
  yellow: "#ffe14d",
  black: "#111111",
};

const CUE_BACKGROUND: Record<SubtitleBackground, string> = {
  none: "transparent",
  soft: "rgb(0 0 0 / 55%)",
  solid: "#000000",
};

/** CSS custom properties to set on the element that wraps the <video>. The cue rule in VideoPlayerSurface.module.css
 *  reads them, so changing a preference is a style change, not a re-render of the track elements. */
export function cueStyle(preferences: SubtitlePreferences): Record<string, string> {
  const scale = CUE_SCALE[preferences.size];
  return {
    "--cue-size": `${scale}em`,
    "--cue-scale": String(scale),
    "--cue-colour": CUE_COLOUR[preferences.colour],
    "--cue-background": CUE_BACKGROUND[preferences.background],
    "--cue-edge": cueEdge(preferences.edge),
  };
}

/** `::cue` accepts text-shadow but not `text-stroke`, so an outline is four offset shadows of the same colour — which
 *  is what every subtitle renderer does under the name "outline" anyway. */
function cueEdge(edge: SubtitleEdge): string {
  const ink = "rgb(0 0 0 / 95%)";
  switch (edge) {
    case "outline":
      return `-1px -1px 0 ${ink}, 1px -1px 0 ${ink}, -1px 1px 0 ${ink}, 1px 1px 0 ${ink}, 0 0 4px ${ink}`;
    case "shadow":
      return `0 2px 4px ${ink}`;
    case "none":
      return "none";
  }
}
