// features/settings/api/sync.ts — the `kosync` section: progress sync with KOReader devices.
//
// Three providers, per `appconfig`'s validation:
//   built-in  this server is the sync server; devices point at it
//   external  devices sync to someone else's KOReader sync server, and this one follows along
//   disabled  no sync at all
// `disabled` and `enabled` are two different flags in the same section and the server honours both, so the panel
// presents the provider choice as the real switch it is rather than adding a fourth state that only exists here.
import { fetchSettings, updateSettingsSection, type SettingsApplyResult, type SettingsDocument } from "./settings";
import { secretConfigured, typedSecret } from "./secrets";

export type SyncProvider = "built-in" | "external" | "disabled";

export const SYNC_PROVIDERS: { id: SyncProvider; label: string }[] = [
  { id: "built-in", label: "This server" },
  { id: "external", label: "Another server" },
  { id: "disabled", label: "Off" },
];

export interface SyncSettings {
  enabled: boolean;
  provider: SyncProvider;
  serverUrl: string;
  username: string;
  passwordConfigured: boolean;
  /** The account the server actually syncs as — `kosync.effective_username`, which can differ from `username`
   *  when the server is authoritative for its own accounts. */
  effectiveUsername: string;
  allowRegistration: boolean;
  webReaderBridge: boolean;
  database: string;
}

function section(settings: SettingsDocument): Record<string, unknown> {
  const raw = settings.kosync;
  return typeof raw === "object" && raw !== null && !Array.isArray(raw) ? (raw as Record<string, unknown>) : {};
}

export function readSync(settings: SettingsDocument): SyncSettings {
  const kosync = section(settings);
  const provider =
    kosync.provider === "external" || kosync.provider === "disabled" ? (kosync.provider as SyncProvider) : "built-in";
  return {
    enabled: kosync.enabled === true,
    provider,
    serverUrl: typeof kosync.server_url === "string" ? kosync.server_url : "",
    username: typeof kosync.username === "string" ? kosync.username : "",
    passwordConfigured: secretConfigured(kosync, "password_configured"),
    effectiveUsername: typeof kosync.effective_username === "string" ? kosync.effective_username : "",
    allowRegistration: kosync.allow_registration === true,
    // Absent means on: the server's own default, expressed as a `*bool` in Go precisely so "unset" is not "off".
    webReaderBridge: kosync.web_reader_bridge !== false,
    database: typeof kosync.database === "string" ? kosync.database : "",
  };
}

export async function fetchSync(): Promise<SyncSettings> {
  return readSync(await fetchSettings());
}

export interface SyncUpdate {
  enabled: boolean;
  provider: SyncProvider;
  serverUrl: string;
  username: string;
  allowRegistration: boolean;
  webReaderBridge: boolean;
  /** Only when the reader typed a new one; an untouched field must not be sent at all. */
  newPassword: string;
}

export async function saveSync(update: SyncUpdate): Promise<SettingsApplyResult> {
  return updateSettingsSection("kosync", {
    enabled: update.enabled,
    provider: update.provider,
    server_url: update.serverUrl.trim(),
    username: update.username.trim(),
    allow_registration: update.allowRegistration,
    web_reader_bridge: update.webReaderBridge,
    ...typedSecret("password", update.newPassword),
  });
}
