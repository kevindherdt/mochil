// features/settings/api/narration.ts — the `narration` section (a local Pocket TTS voice runtime) and the read-only
// /api/narration/status that says whether it is actually running.
//
// The status endpoint is what makes this panel honest: a saved configuration says nothing about whether the runtime
// is installed, and "narration doesn't work" almost always means the runtime is absent rather than misconfigured. So
// the panel shows Enabled (the module switch), Available (the runtime is installed) and Loaded (the model is in
// memory) as three separate facts instead of one vague tick.
//
// Paths (`python`, `cache`, `voice_directory`) are shown but not edited. They are install details with server-side
// defaults (`appconfig` fills them from the bundled runtime and the cache directory), and a hand-typed path that is
// wrong fails at synthesis time with an error nobody would connect back to this form. The panel shows the effective
// values so they can be checked, which is what they are actually needed for.
import { z } from "zod";
import { request } from "../../../lib/http";
import { fetchSettings, updateSettingsSection, type SettingsApplyResult, type SettingsDocument } from "./settings";
import { secretConfigured, typedSecret } from "./secrets";

export interface NarrationSettings {
  offline: boolean;
  threads: number;
  idleSeconds: number;
  tokenConfigured: boolean;
  python: string;
  cache: string;
  voiceDirectory: string;
}

function section(settings: SettingsDocument): Record<string, unknown> {
  const raw = settings.narration;
  return typeof raw === "object" && raw !== null && !Array.isArray(raw) ? (raw as Record<string, unknown>) : {};
}

function number(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

export function readNarration(settings: SettingsDocument): NarrationSettings {
  const narration = section(settings);
  return {
    offline: narration.offline === true,
    // 0 means "let the runtime decide" for both of these, which is the server's own default.
    threads: number(narration.threads, 0),
    idleSeconds: number(narration.idle_seconds, 0),
    tokenConfigured: secretConfigured(narration, "narration_token_configured"),
    python: typeof narration.python === "string" ? narration.python : "",
    cache: typeof narration.cache === "string" ? narration.cache : "",
    voiceDirectory: typeof narration.voice_directory === "string" ? narration.voice_directory : "",
  };
}

export async function fetchNarration(): Promise<NarrationSettings> {
  return readNarration(await fetchSettings());
}

export interface NarrationUpdate {
  offline: boolean;
  threads: number;
  idleSeconds: number;
  /** Only when the reader typed one — see api/secrets.ts. */
  newToken: string;
}

export async function saveNarration(update: NarrationUpdate): Promise<SettingsApplyResult> {
  return updateSettingsSection("narration", {
    offline: update.offline,
    threads: update.threads,
    idle_seconds: update.idleSeconds,
    ...typedSecret("token", update.newToken),
  });
}

const voiceSchema = z.object({ name: z.string(), custom: z.boolean().optional() });

export const narrationStatusSchema = z.object({
  enabled: z.boolean(),
  available: z.boolean(),
  loaded: z.boolean(),
  voice: z.string(),
  voices: z.array(voiceSchema),
  listen: z.boolean(),
  manage: z.boolean(),
});

export type NarrationStatus = z.infer<typeof narrationStatusSchema>;

export async function fetchNarrationStatus(): Promise<NarrationStatus> {
  return narrationStatusSchema.parse(await request<unknown>("/api/narration/status"));
}

/** One sentence for the three runtime facts, because they are only meaningful together: a module can be on with no
 *  runtime installed, and a runtime can be installed without the model being loaded yet. */
export function describeRuntime(status: NarrationStatus): string {
  if (!status.enabled) return "The narration module is switched off.";
  if (!status.available) return "The voice runtime is not installed, so nothing can be read aloud yet.";
  if (!status.loaded) return "The runtime is installed; the voice loads the first time something is read aloud.";
  return `Ready${status.voice === "" ? "" : ` — reading with ${status.voice}`}.`;
}
