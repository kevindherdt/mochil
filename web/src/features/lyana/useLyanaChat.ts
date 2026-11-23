// features/lyana/useLyanaChat.ts — the real `/api/hermes/chat` wire protocol (plan ui-overhaul-plan.md §B4),
// ported from the legacy static/hermes-activity.js + hermes.js send() rather than reinvented: real Server-Sent
// Events (`data: {...}\n\n`, terminated by `data: [DONE]`), OpenAI-shaped deltas
// (`choices[0].delta.content`), plus two Mochil-specific named events this server also sends on the same
// stream: `event: hermes.cards` (library items she offers to play) and `event: hermes.error` (a mid-stream
// failure). `event: hermes.waiting`/`hermes.tool.progress` (the legacy UI's step-by-step "searching the
// web / reading a page" list) are deliberately NOT parsed here — this slice ships plain streaming text and
// cards, and shows a single generic "thinking…" status instead of the full step list; that is a real
// left-out-on-purpose gap (recorded in docs/rewrite-progress.md), not a silent drop, matching how
// AudioDetailScreen documents its own left-out features instead of faking them.
import { useCallback, useEffect, useRef, useState } from "react";
import { openChat } from "./api/chats";

export interface LyanaCard {
  kind: string;
  library: string;
  id: string;
  title: string;
  subtitle?: string | undefined;
  streamUrl: string;
  artworkUrl?: string | undefined;
}

/** One file riding with a message. `data` is base64 without the data: prefix (the server decodes it). */
export interface ChatAttachment {
  name: string;
  media_type: string;
  data: string;
}

export interface LyanaMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  cards?: LyanaCard[] | undefined;
  /** True only for the in-flight assistant reply, so the UI can show a cursor/status instead of "nothing yet". */
  streaming?: boolean | undefined;
  error?: string | undefined;
  /** Names of files that rode with a user message, so the bubble can show what was sent. */
  attachmentNames?: string[] | undefined;
}

const SESSION_KEY = "mochil-hermes-current";

/** `crypto.randomUUID()` requires a secure context (https, or localhost) — it throws on a plain LAN `http://`
    origin, which is exactly how this server is reached day to day (confirmed live: `crypto.randomUUID is not a
    function` on a plain `http://` address, the same secure-context gotcha the web-ui skill already documents
    for service workers). `crypto.getRandomValues` has no such restriction, so fall back to it rather than to
    `Math.random()` alone. */
function randomId(): string {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

/** Mirrors hermes.js's own id: lowercase, 8-48 chars, matching the server's hermesChatIDPattern. */
function newSessionId(): string {
  return `web-${randomId()}`.slice(0, 40);
}

function readSession(): string {
  try {
    const saved = window.localStorage.getItem(SESSION_KEY);
    if (saved !== null && saved !== "") return saved;
  } catch {
    // Storage can be unavailable (private browsing) — a session that doesn't persist across reloads is still a
    // working session for this visit, so fall through to a fresh one instead of failing to chat at all.
  }
  const fresh = newSessionId();
  try {
    window.localStorage.setItem(SESSION_KEY, fresh);
  } catch {
    // Same as above: persistence is a convenience.
  }
  return fresh;
}

interface StreamCardsPayload {
  cards?: unknown;
}

function parseCards(raw: unknown): LyanaCard[] {
  if (!Array.isArray(raw)) return [];
  const cards: LyanaCard[] = [];
  for (const entry of raw) {
    if (typeof entry !== "object" || entry === null) continue;
    const record = entry as Record<string, unknown>;
    if (typeof record.title !== "string" || typeof record.stream_url !== "string") continue;
    cards.push({
      kind: typeof record.kind === "string" ? record.kind : "",
      library: typeof record.library === "string" ? record.library : "",
      id: typeof record.id === "string" ? record.id : "",
      title: record.title,
      subtitle: typeof record.subtitle === "string" ? record.subtitle : undefined,
      streamUrl: record.stream_url,
      artworkUrl: typeof record.artwork_url === "string" ? record.artwork_url : undefined,
    });
  }
  return cards;
}

export interface LyanaChatState {
  messages: LyanaMessage[];
  busy: boolean;
  /** The chat this screen is talking in — the Hermes session id, which is also what `/api/hermes/chats` lists. */
  session: string;
  /** True while an earlier chat is being loaded into the transcript. */
  loading: boolean;
  send: (text: string, attachments?: ChatAttachment[]) => Promise<void>;
  stop: () => void;
  /** Reattaches to a chat that already exists, loading its transcript. */
  resume: (id: string) => Promise<void>;
  /** Starts a fresh chat and remembers it for the next visit. */
  startNew: () => void;
}

/** Writes the current chat id, so a reload continues where the person left off instead of showing an empty screen. */
function rememberSession(id: string) {
  try {
    window.localStorage.setItem(SESSION_KEY, id);
  } catch {
    // Storage is a convenience; a chat still works for this visit without it.
  }
}

export function useLyanaChat(): LyanaChatState {
  const [messages, setMessages] = useState<LyanaMessage[]>([]);
  const [busy, setBusy] = useState(false);
  const [session, setSession] = useState<string>(() => readSession());
  const [loading, setLoading] = useState(false);
  const sessionRef = useRef<string>(session);
  const abortRef = useRef<AbortController | null>(null);

  sessionRef.current = session;

  // On a reload the screen used to come back empty even though the session id survived: the id said which conversation
  // this is, and nothing ever asked the server what was in it. Load it once, quietly — a chat with no history answers
  // with an empty list, so this costs one request and removes the "my conversation vanished" surprise.
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const history = await openChat(sessionRef.current);
        // Anything typed while the history was loading stays: the transcript is only replaced when nothing has happened.
        if (!cancelled) setMessages((current) => (current.length === 0 ? history : current));
      } catch {
        // A server that cannot list history still chats; an empty transcript is the same as a new conversation.
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const resume = useCallback(async (id: string) => {
    abortRef.current?.abort();
    setLoading(true);
    try {
      const history = await openChat(id);
      sessionRef.current = id;
      setSession(id);
      rememberSession(id);
      setMessages(history);
    } finally {
      setLoading(false);
    }
  }, []);

  const startNew = useCallback(() => {
    abortRef.current?.abort();
    const fresh = newSessionId();
    sessionRef.current = fresh;
    setSession(fresh);
    rememberSession(fresh);
    setMessages([]);
  }, []);

  const stop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const send = useCallback(async (text: string, attachments?: ChatAttachment[]) => {
    const trimmed = text.trim();
    // A file with no words is still a message — a screenshot and "look at this" are different asks, and the first
    // one should not be blocked for having no text.
    if ((trimmed === "" && (attachments === undefined || attachments.length === 0)) || busy) return;
    const userMessage: LyanaMessage = {
      id: randomId(),
      role: "user",
      text: trimmed,
      ...(attachments === undefined || attachments.length === 0 ? {} : { attachmentNames: attachments.map((file) => file.name) }),
    };
    const replyId = randomId();
    setMessages((previous) => [...previous, userMessage, { id: replyId, role: "assistant", text: "", streaming: true }]);
    setBusy(true);
    const controller = new AbortController();
    abortRef.current = controller;

    let reply = "";
    let cards: LyanaCard[] | undefined;
    let failure = "";
    try {
      const response = await fetch("/api/hermes/chat", {
        method: "POST",
        credentials: "same-origin",
        signal: controller.signal,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          session: sessionRef.current,
          message: trimmed,
          // Absent rather than empty when there is nothing attached, so a text-only message keeps the exact request
          // shape it had before attachments existed.
          ...(attachments === undefined || attachments.length === 0 ? {} : { attachments }),
        }),
      });
      if (!response.ok || response.body === null) {
        const body = await response.json().catch(() => ({}) as { error?: string });
        throw new Error(body.error ?? `Request failed (${response.status})`);
      }
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let pending = "";
      let event = "";
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        pending += decoder.decode(value, { stream: true });
        let cut: number;
        while ((cut = pending.indexOf("\n")) >= 0) {
          const raw = pending.slice(0, cut);
          pending = pending.slice(cut + 1);
          const line = raw.trim();
          if (line === "") {
            event = "";
            continue;
          }
          if (line.startsWith("event:")) {
            event = line.slice(6).trim();
            continue;
          }
          if (!line.startsWith("data:")) continue;
          const data = line.slice(5).trim();
          if (data === "" || data === "[DONE]") continue;
          let body: unknown;
          try {
            body = JSON.parse(data);
          } catch {
            continue;
          }
          if (event === "hermes.cards") {
            cards = parseCards((body as StreamCardsPayload).cards);
            setMessages((previous) => previous.map((message) => (message.id === replyId ? { ...message, cards } : message)));
            continue;
          }
          if (event === "hermes.error") {
            const errorBody = body as { error?: unknown };
            failure = typeof errorBody.error === "string" ? errorBody.error : "The answer was interrupted";
            continue;
          }
          const delta = (body as { choices?: { delta?: { content?: unknown } }[] }).choices?.[0]?.delta?.content;
          if (typeof delta === "string" && delta !== "") {
            reply += delta;
            setMessages((previous) => previous.map((message) => (message.id === replyId ? { ...message, text: reply } : message)));
          }
        }
      }
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        failure = "";
      } else {
        failure = error instanceof Error ? error.message : "Lyana did not answer";
      }
    }
    setMessages((previous) =>
      previous.map((message) =>
        message.id === replyId
          ? { ...message, streaming: false, error: failure !== "" ? failure : undefined, text: message.text }
          : message,
      ),
    );
    abortRef.current = null;
    setBusy(false);
  }, [busy]);

  return { messages, busy, session, loading, send, stop, resume, startNew };
}
