// features/settings/api/providers.ts — the `metadata_providers` section: the two online catalogue tokens.
//
// Both are secrets and both behave the same way: GET returns `hardcover_token_configured` rather than the token, and
// a save without the field keeps whatever is stored (`settings/preserveProviderSecrets`). So there is deliberately
// no "remove the token" control here — the server has no way to clear one, and a button that pretends otherwise
// would be lying. Replacing is what this panel does; the panel says so.
//
// Which provider a library uses is not set here: each library derives it from its kind (`appconfig`'s
// `automaticMetadataProvider` — Hardcover for publications when a token exists, otherwise Open Library; TMDB for
// video), which is why a token without a matching library simply does nothing.
import { fetchSettings, updateSettingsSection, type SettingsApplyResult, type SettingsDocument } from "./settings";
import { secretConfigured, typedSecret } from "./secrets";

export interface ProviderSettings {
  hardcoverConfigured: boolean;
  tmdbConfigured: boolean;
}

function section(settings: SettingsDocument): Record<string, unknown> {
  const raw = settings.metadata_providers;
  return typeof raw === "object" && raw !== null && !Array.isArray(raw) ? (raw as Record<string, unknown>) : {};
}

export function readProviders(settings: SettingsDocument): ProviderSettings {
  const providers = section(settings);
  return {
    hardcoverConfigured: secretConfigured(providers, "hardcover_token_configured"),
    tmdbConfigured: secretConfigured(providers, "tmdb_token_configured"),
  };
}

export async function fetchProviders(): Promise<ProviderSettings> {
  return readProviders(await fetchSettings());
}

export async function saveProviderTokens(input: { hardcover: string; tmdb: string }): Promise<SettingsApplyResult> {
  return updateSettingsSection("metadata_providers", {
    ...typedSecret("hardcover_token", input.hardcover),
    ...typedSecret("tmdb_token", input.tmdb),
  });
}
