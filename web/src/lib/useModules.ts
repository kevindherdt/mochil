// lib/useModules — /api/modules, one of the 8 contract-fixtured endpoints. Both the shell (does the dock get a
// Lyana place?) and the library header (does the AI button exist?) ask this, so it lives in lib/ rather than in
// either feature.
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { request } from "./http";

const moduleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  enabled: z.boolean(),
  default_on: z.boolean().nullish(),
  // The live server sends `"kinds": null` for modules that apply to no library kind (narration, hermes, editor…),
  // while the contract fixture simply omits the field. `.optional()` alone rejected the null and made this whole
  // query throw — which silently hid the AI button. `.nullish()` accepts both shapes.
  kinds: z.array(z.string()).nullish(),
});

export type ModuleInfo = z.infer<typeof moduleSchema>;

/** The chat/AI module's id on the real server is `hermes` (name "Hermes chat", surfaced as Lyana AI). */
export const AI_MODULE_ID = "hermes";

export async function fetchModules(): Promise<ModuleInfo[]> {
  const raw = await request<unknown>("/api/modules");
  return z.array(moduleSchema).parse(raw);
}

/** Modules only change in Settings, so this is cached for the session instead of refetched per screen. */
export function useModules(): ModuleInfo[] {
  const query = useQuery({ queryKey: ["modules"], queryFn: fetchModules, staleTime: Infinity, retry: false });
  return query.data ?? [];
}

/** The library editor is a screen behind a module: off means no editor, not a broken one. */
export function useEditorEnabled(): boolean {
  return useModules().some((module) => module.id === "editor" && module.enabled);
}

/** Whether the AI/Lyana module is enabled — the one thing the chrome needs to know. */
export function useAiEnabled(): boolean {
  return useModules().some((module) => module.id === AI_MODULE_ID && module.enabled);
}

/** Same check as `useAiEnabled`, but distinguishes "the query hasn't answered yet" from "it answered and the
    module is off" — `useAiEnabled()` collapses both to `false`, which is fine for a nav icon that just doesn't
    render for a moment, but was a real bug for Lyana's full-page gate: every fresh page load (a login redirect
    always counts) flashed "Lyana isn't switched on" for the one render before the query resolved. */
export function useAiModuleState(): "loading" | "enabled" | "disabled" {
  const query = useQuery({ queryKey: ["modules"], queryFn: fetchModules, staleTime: Infinity, retry: false });
  if (query.data === undefined) return query.isError ? "disabled" : "loading";
  return query.data.some((module) => module.id === AI_MODULE_ID && module.enabled) ? "enabled" : "disabled";
}
