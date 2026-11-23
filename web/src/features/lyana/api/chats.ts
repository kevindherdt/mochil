// features/lyana/api/chats.ts — the conversations the server already keeps.
//
// Hermes holds one session per chat, and Mochil exposes them per account:
//
//   GET    /api/hermes/chats        → { chats: [{ id, title, updated, count }] }, newest first
//   GET    /api/hermes/chats/<id>   → { messages: [{ role, text, cards? }] }
//   PATCH  /api/hermes/chats/<id>   { title }
//   DELETE /api/hermes/chats/<id>
//
// The ids are the same ones `POST /api/hermes/chat` takes as `session`, which is what makes "continue a chat" work
// without inventing a store here: reopening a chat reattaches to the session Hermes already has the history for.
import { z } from "zod";
import { request } from "../../../lib/http";
import type { LyanaCard, LyanaMessage } from "../useLyanaChat";

const summarySchema = z.looseObject({
  id: z.string().optional(),
  title: z.string().optional(),
  updated: z.number().optional(),
  count: z.number().optional(),
});

export interface ChatSummary {
  id: string;
  title: string;
  /** Unix seconds from Hermes; 0 when it did not say. */
  updated: number;
  count: number;
}

export async function listChats(): Promise<ChatSummary[]> {
  const raw = await request<{ chats?: unknown }>("/api/hermes/chats");
  if (!Array.isArray(raw.chats)) return [];
  const chats: ChatSummary[] = [];
  for (const entry of raw.chats) {
    const parsed = summarySchema.safeParse(entry);
    if (!parsed.success || parsed.data.id === undefined || parsed.data.id === "") continue;
    chats.push({
      id: parsed.data.id,
      title: parsed.data.title ?? "New chat",
      updated: parsed.data.updated ?? 0,
      count: parsed.data.count ?? 0,
    });
  }
  return chats;
}

const cardSchema = z.looseObject({
  kind: z.string().optional(),
  library: z.string().optional(),
  id: z.string().optional(),
  title: z.string().optional(),
  subtitle: z.string().optional(),
  stream_url: z.string().optional(),
  artwork_url: z.string().optional(),
});

function toCards(raw: unknown): LyanaCard[] | undefined {
  if (!Array.isArray(raw) || raw.length === 0) return undefined;
  const cards: LyanaCard[] = [];
  for (const entry of raw) {
    const parsed = cardSchema.safeParse(entry);
    if (!parsed.success || parsed.data.title === undefined || parsed.data.stream_url === undefined) continue;
    cards.push({
      kind: parsed.data.kind ?? "",
      library: parsed.data.library ?? "",
      id: parsed.data.id ?? "",
      title: parsed.data.title,
      subtitle: parsed.data.subtitle,
      streamUrl: parsed.data.stream_url,
      artworkUrl: parsed.data.artwork_url,
    });
  }
  return cards.length === 0 ? undefined : cards;
}

const messageSchema = z.looseObject({
  role: z.string().optional(),
  text: z.string().optional(),
  cards: z.unknown().optional(),
});

/**
 * Loads one chat's transcript. Ids are made unique against what is already on screen: a reopened chat and a live one can
 * otherwise both hold "message 0", and React would reuse the wrong bubble.
 */
export async function openChat(id: string): Promise<LyanaMessage[]> {
  const raw = await request<{ messages?: unknown }>(`/api/hermes/chats/${encodeURIComponent(id)}`);
  if (!Array.isArray(raw.messages)) return [];
  const messages: LyanaMessage[] = [];
  for (const entry of raw.messages) {
    const parsed = messageSchema.safeParse(entry);
    if (!parsed.success) continue;
    const role = parsed.data.role === "user" ? "user" : parsed.data.role === "assistant" ? "assistant" : null;
    const text = parsed.data.text ?? "";
    if (role === null || text.trim() === "") continue;
    messages.push({
      id: `${id}-${messages.length}`,
      role,
      text,
      cards: toCards(parsed.data.cards),
    });
  }
  return messages;
}

export async function renameChat(id: string, title: string): Promise<void> {
  await request<unknown>(`/api/hermes/chats/${encodeURIComponent(id)}`, { method: "PATCH", body: { title } });
}

export async function deleteChat(id: string): Promise<void> {
  await request<unknown>(`/api/hermes/chats/${encodeURIComponent(id)}`, { method: "DELETE" });
}
