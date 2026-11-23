// features/lyana/api/companion.ts — the AI companion's own settings, all of them already on the server.
//
// Kevin's list: "no settings icon for the ai page to control the voice, hands free talking ... no way to change the
// avatar, also the name lyana shouldnt be hardcoded". Three of those have real endpoints that the rewrite simply
// never called:
//
//   GET  /api/narration/status        → { enabled, available, loaded, voice, voices[], listen, manage }
//                                       `voices` is the pickable list, `listen` says whether speech recognition is
//                                       actually set up (the mic must not offer itself when it is not).
//   PUT  /api/narration/default-voice  { voice } — the voice used when speaking replies.
//   GET  /api/avatar/models           → { models[], selected, core, voices{model:voice}, scene, frames, image }
//   PUT  /api/avatar/select           { model, voice } — pick the avatar *and* which voice that avatar uses.
//   POST /api/narration/transcribe    raw audio bytes → { text } (or { text: "", no_speech: true })
//
// The name is the one piece with no server-side store: `/api/settings` is the server's config document and has no
// assistant section, so the name lives in this browser (see useCompanion) rather than being invented into it.
import { request } from "../../../lib/http";

export interface NarrationVoice {
  id: string;
  name: string;
  language?: string;
  custom?: boolean;
}

export interface NarrationStatus {
  enabled: boolean;
  available: boolean;
  loaded: boolean;
  /** Currently selected default voice id. */
  voice: string;
  voices: NarrationVoice[];
  /** Whether speech recognition (the mic) is set up on this server. */
  listen: boolean;
  /** Administrators may change the default voice; everyone else can only use it. */
  manage: boolean;
}

export async function fetchNarrationStatus(): Promise<NarrationStatus> {
  const raw = await request<Partial<NarrationStatus>>("/api/narration/status");
  return {
    enabled: raw.enabled === true,
    available: raw.available === true,
    loaded: raw.loaded === true,
    voice: raw.voice ?? "",
    voices: Array.isArray(raw.voices) ? raw.voices : [],
    listen: raw.listen === true,
    manage: raw.manage === true,
  };
}

export async function setDefaultVoice(voice: string): Promise<void> {
  await request<unknown>("/api/narration/default-voice", { method: "PUT", body: { voice } });
}

export interface AvatarModel {
  id: string;
  name?: string;
  model?: string;
}

export interface AvatarState {
  models: AvatarModel[];
  selected: string;
  /** model id → the voice that model speaks with. */
  voices: Record<string, string>;
  /** Whether the runtime that draws the avatar is present at all. */
  core: boolean;
}

export async function fetchAvatarState(): Promise<AvatarState> {
  const raw = await request<Partial<AvatarState>>("/api/avatar/models");
  return {
    models: Array.isArray(raw.models) ? raw.models : [],
    selected: raw.selected ?? "",
    voices: typeof raw.voices === "object" && raw.voices !== null ? raw.voices : {},
    core: raw.core === true,
  };
}

/** Picks the avatar, and optionally which voice it speaks with — one call, because they are one choice. */
export async function selectAvatar(model: string, voice?: string): Promise<void> {
  await request<unknown>("/api/avatar/select", {
    method: "PUT",
    body: voice === undefined ? { model } : { model, voice },
  });
}

/** Sends a raw recording (webm/opus from MediaRecorder) and returns what was said. Empty text means "no speech". */
export async function transcribe(recording: Blob): Promise<string> {
  const response = await fetch("/api/narration/transcribe", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": recording.type || "audio/webm" },
    body: recording,
  });
  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    throw new Error(detail === "" ? `Could not transcribe (${response.status})` : detail);
  }
  const payload = (await response.json()) as { text?: string };
  return payload.text ?? "";
}
