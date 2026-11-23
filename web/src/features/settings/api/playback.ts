// features/settings/api/playback.ts — the `playback` section of the settings document, plus the read-only
// /api/playback-policy the players actually obey.
//
// Two different things live in that one section and it matters not to blur them:
//   * `wifi_quality` / `mobile_quality` are the *ceiling* a player may stream at. `/api/playback-policy` hands them
//     to the client (defaulting to maximum/high), and that endpoint is the one a player reads — this panel writes
//     the same values into the config the endpoint reads from.
//   * `audio_language` / `subtitle_language` are server-wide *defaults* for which audio and subtitle track a video
//     should start on. A per-show choice overrides them (internal/video's PreferenceStore), so these are defaults,
//     not locks — the UI says so instead of implying it forces every title.
import { z } from "zod";
import { request } from "../../../lib/http";
import { fetchSettings, updateSettingsSection, type SettingsApplyResult, type SettingsDocument } from "./settings";

export type PlaybackQuality = "maximum" | "high" | "medium" | "low";

export const PLAYBACK_QUALITIES: { id: PlaybackQuality; label: string }[] = [
  { id: "maximum", label: "Maximum" },
  { id: "high", label: "High" },
  { id: "medium", label: "Medium" },
  { id: "low", label: "Low" },
];

export interface PlaybackSettings {
  audioLanguage: string;
  subtitleLanguage: string;
  wifiQuality: PlaybackQuality;
  mobileQuality: PlaybackQuality;
}

/** The server's own defaults, mirrored so the panel shows what a fresh config does (`appconfig` fills these in when
 *  the file omits them, and `playback_policy_http.go` falls back to the same pair). */
export const PLAYBACK_DEFAULTS: PlaybackSettings = {
  audioLanguage: "",
  subtitleLanguage: "",
  wifiQuality: "maximum",
  mobileQuality: "high",
};

const qualitySchema = z.enum(["maximum", "high", "medium", "low"]);

export function readPlayback(settings: SettingsDocument): PlaybackSettings {
  const raw = settings.playback;
  const section = typeof raw === "object" && raw !== null && !Array.isArray(raw) ? (raw as Record<string, unknown>) : {};
  const quality = (value: unknown, fallback: PlaybackQuality): PlaybackQuality =>
    qualitySchema.safeParse(value).success ? (value as PlaybackQuality) : fallback;
  return {
    audioLanguage: typeof section.audio_language === "string" ? section.audio_language : PLAYBACK_DEFAULTS.audioLanguage,
    subtitleLanguage:
      typeof section.subtitle_language === "string" ? section.subtitle_language : PLAYBACK_DEFAULTS.subtitleLanguage,
    wifiQuality: quality(section.wifi_quality, PLAYBACK_DEFAULTS.wifiQuality),
    mobileQuality: quality(section.mobile_quality, PLAYBACK_DEFAULTS.mobileQuality),
  };
}

export async function fetchPlayback(): Promise<PlaybackSettings> {
  return readPlayback(await fetchSettings());
}

/** Writes all four fields. Language codes are lowercased and trimmed because that is what the server does with them
 *  (`internal/video`'s normalisePreferences) — sending "EN" and then showing it back unchanged would be showing a
 *  value the server never stored. */
export async function savePlayback(settings: PlaybackSettings): Promise<SettingsApplyResult> {
  return updateSettingsSection("playback", {
    audio_language: settings.audioLanguage.trim().toLowerCase(),
    subtitle_language: settings.subtitleLanguage.trim().toLowerCase(),
    wifi_quality: settings.wifiQuality,
    mobile_quality: settings.mobileQuality,
  });
}

/** GET /api/playback-policy — what a player is actually told, after the server has applied its own fallbacks. Shown
 *  next to the settings so "what I typed" and "what players get" can be compared rather than assumed. */
const policySchema = z.object({ wifi_quality: z.string(), mobile_quality: z.string() });

export async function fetchPlaybackPolicy(): Promise<{ wifiQuality: string; mobileQuality: string }> {
  const raw = policySchema.parse(await request<unknown>("/api/playback-policy"));
  return { wifiQuality: raw.wifi_quality, mobileQuality: raw.mobile_quality };
}
