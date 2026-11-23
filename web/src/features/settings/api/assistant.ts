// features/settings/api/assistant.ts — where the assistant's endpoint lives.
//
// The assistant talks to whatever endpoint the server is pointed at: a Hermes gateway, or anything that answers
// `/v1/chat/completions`. That endpoint is the server's config document (`/api/settings`), and until now the only way to
// set it was to edit `config.json` by hand — fine for the person who wrote Mochil, unreasonable for anyone else.
//
// The document is read and written through the same `/api/settings` the other panels use. `hermes.key` is never sent to
// the browser: the server reports `key_configured` instead, and a blank key on save keeps the stored one, exactly like
// the narration token and the metadata provider tokens.
import { request } from "../../../lib/http";

export interface AssistantSettings {
  /** Base address, e.g. `http://host:8642/v1` or `http://host:8642/p/companion`. */
  url: string;
  model: string;
  /** What the assistant is called. */
  name: string;
  /** Whether a key is already stored on the server (the key itself is never sent here). */
  keyConfigured: boolean;
  /** The rest of the document, so a save writes back everything it read. */
  document: Record<string, unknown>;
}

function section(document: Record<string, unknown>): Record<string, unknown> {
  const hermes = document["hermes"];
  return typeof hermes === "object" && hermes !== null && !Array.isArray(hermes)
    ? { ...(hermes as Record<string, unknown>) }
    : {};
}

export async function fetchAssistantSettings(): Promise<AssistantSettings> {
  const document = await request<Record<string, unknown>>("/api/settings");
  const hermes = section(document);
  return {
    url: typeof hermes["url"] === "string" ? hermes["url"] : "",
    model: typeof hermes["model"] === "string" ? hermes["model"] : "",
    name: typeof hermes["name"] === "string" ? hermes["name"] : "",
    keyConfigured: hermes["key_configured"] === true,
    document,
  };
}

export interface AssistantChange {
  url: string;
  model: string;
  name: string;
  /** Empty means "keep the key that is already stored". */
  key?: string;
}

export async function saveAssistantSettings(current: AssistantSettings, change: AssistantChange): Promise<void> {
  const hermes = section(current.document);
  hermes["url"] = change.url.trim();
  hermes["model"] = change.model.trim();
  if (change.name.trim() !== "") hermes["name"] = change.name.trim();
  delete hermes["key_configured"];
  if (change.key !== undefined && change.key.trim() !== "") {
    hermes["key"] = change.key.trim();
  } else {
    // Absent rather than empty: the server keeps the stored key when the field is missing, and treats an explicit ""
    // as "no key", which would silently break a working setup.
    delete hermes["key"];
  }
  await request<unknown>("/api/settings", {
    method: "PUT",
    body: { ...current.document, hermes },
  });
}

export interface AssistantTest {
  ok: boolean;
  message: string;
}

/** Asks the server to try the endpoint it is configured with, and says what came back. */
export async function testAssistant(): Promise<AssistantTest> {
  try {
    const result = await request<{ ok?: boolean; error?: string; model?: string; reply?: string }>("/api/hermes/test", {
      method: "POST",
      body: {},
    });
    if (result.ok === false) return { ok: false, message: result.error ?? "The endpoint answered, but not with a reply." };
    const model = result.model ?? "";
    return { ok: true, message: model === "" ? "The endpoint answered." : `The endpoint answered as ${model}.` };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : "The endpoint could not be reached." };
  }
}
