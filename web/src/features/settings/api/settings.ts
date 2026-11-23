// features/settings/api/settings.ts — GET/PUT /api/settings (internal/web/settings_http.go).
//
// The document is deliberately NOT modelled as a zod schema with named fields. `settingssvc.Read` returns the
// server's whole config (appearance, cover cache, hermes, kosync, libraries, listen, metadata providers, modules,
// narration, page cache, playback, public URL, scan interval …) and PUT replaces it wholesale. A hand-written
// schema would have to be kept in lock-step with internal/settings, and the failure mode when it drifts is the
// worst possible one: a field the web app doesn't know about gets dropped on the next save, silently, in a config
// file the server needs. So the document is carried as `unknown` records and mutated by key — the round trip
// preserves every field the web app has never heard of, which is the only safe way to write a whole-document API
// from a client that only wants to change one thing.
//
// This is also how the legacy panel does it (static/modules-settings.js's `write()` copies `target.modules` and
// sets only the touched ids), so the two UIs can't disagree about what a save means.
import { request, HttpError } from "../../../lib/http";

export type SettingsDocument = Record<string, unknown>;

export async function fetchSettings(): Promise<SettingsDocument> {
  const raw = await request<unknown>("/api/settings");
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    throw new HttpError(500, "/api/settings", "settings document was not an object");
  }
  return raw as SettingsDocument;
}

/** Writes the whole settings document and returns the server's apply report.
 *
 *  `PUT /api/settings` is not a plain save: the live config is re-decoded, changed libraries are opened and scanned,
 *  the handler is swapped, and the answer says what actually happened (`saved`, `applied`, `libraries`,
 *  `libraries_scanned`, `libraries_reused`, `restart_required`). Callers that show the result must show that answer,
 *  not an assumption that saving worked. */
export interface SettingsApplyResult {
  saved: boolean;
  applied: boolean;
  libraries: number;
  libraries_scanned?: number;
  libraries_reused?: number;
  restart_required?: string[];
}

export async function putSettings(document: SettingsDocument): Promise<SettingsApplyResult> {
  const saved = await request<unknown>("/api/settings", { method: "PUT", body: document });
  if (typeof saved !== "object" || saved === null) {
    return { saved: true, applied: false, libraries: 0 };
  }
  return saved as SettingsApplyResult;
}

/** Merges `patch` into one object section of the document (`playback`, `modules`, …) and saves.
 *
 *  Keys the panel doesn't know about inside the section survive, which matters for `playback` (a future setting) as
 *  much as for `modules`. Read-then-write, for the reason above. */
export async function updateSettingsSection(
  section: string,
  patch: Record<string, unknown>,
): Promise<SettingsApplyResult> {
  const document = await fetchSettings();
  const current = document[section];
  const existing =
    typeof current === "object" && current !== null && !Array.isArray(current) ? (current as Record<string, unknown>) : {};
  return putSettings({ ...document, [section]: { ...existing, ...patch } });
}

/** Switches exactly the given module ids, leaving every other field of the document untouched.
 *
 *  Read-then-write rather than write-then-patch: the server has no partial-update endpoint, and sending a document
 *  the client assembled from a stale cache would revert anything changed in the legacy UI since this tab loaded.
 *  The extra GET is one small request and buys that correctness.
 */
export async function saveModuleSwitches(switches: Record<string, boolean>): Promise<SettingsApplyResult> {
  // Only ids the user actually moved become explicit; the rest keep following their libraries, exactly like the
  // legacy panel's `write()` (which sends `touched` and nothing else).
  return updateSettingsSection("modules", switches);
}

/** POST /api/restart — a module that was switched off is still loaded until the process restarts, so the panel
 *  offers this instead of implying the switch took effect immediately. Answers 202 `{restarting: true}`. */
export async function restartServer(): Promise<void> {
  await request<unknown>("/api/restart", { method: "POST" });
}
