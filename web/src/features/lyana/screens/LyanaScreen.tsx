// features/lyana/screens/LyanaScreen.tsx — the real Lyana AI body (plan ui-overhaul-plan.md §B4). Was a pure
// stub (`<StubScreen title="Lyana AI" />`) until this slice.
//
// Scope, honestly, the same way AudioDetailScreen documents its own gaps: this ships a real, working text chat —
// streaming replies, the "recommended from your library" cards, a message log that survives navigating away and
// back (state lives in the one `useLyanaChat()` call this screen owns; the legacy app persists the session id
// itself via localStorage the same way, so a reload resumes the same Hermes session). Left out on purpose:
// the composer's `+`/mic (features/../ui/Composer.tsx explains why), the per-step "searching the
// web / reading a page" activity list the legacy hermes.js shows (collapsed here into one generic "Thinking…"
// line), suggestion chips, multi-chat history/rename/delete (`/api/hermes/chats`), and the filter row the mock-up
// draws above the log (Books · Audiobooks · Music / All · Manga) — nothing here reads or writes a filter yet, so
// drawing it would be chrome with no effect.
import { useEffect, useRef, useState } from "react";
import { Bot, User } from "lucide-react";
import { Page, Stack, Inline, Text, Heading, Composer, CoverImage, EmptyState } from "../../../ui";
import { useLyanaChat, type ChatAttachment, type LyanaMessage } from "../useLyanaChat";
import { CompanionSheet } from "../CompanionSheet";
import { useCompanionName, useHandsFree, useNarration } from "../useCompanion";
import { transcribe } from "../api/companion";
import { Settings } from "lucide-react";
import { IconButton, type PendingAttachment } from "../../../ui";
import { useAiModuleState } from "../../../lib/useModules";
import { usePlayer } from "../../player/PlayerContext";

function MessageBubble({ message }: { message: LyanaMessage }) {
  const player = usePlayer();
  const isUser = message.role === "user";
  return (
    <Inline gap={8} align="flex-start" justify={isUser ? "flex-end" : "flex-start"}>
      {!isUser && (
        <div style={{ flexShrink: 0, width: 28, height: 28, borderRadius: "var(--radius-pill)", background: "var(--color-raised)", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--color-accent)" }}>
          <Bot size={15} />
        </div>
      )}
      <Stack gap={8} align={isUser ? "flex-end" : "flex-start"}>
        <div
          style={{
            maxWidth: "min(480px, 80vw)",
            padding: "var(--space-8) var(--space-12)",
            borderRadius: "var(--radius-16)",
            background: isUser ? "var(--color-accent)" : "var(--color-raised)",
            color: isUser ? "var(--color-bg)" : "var(--color-text)",
          }}
        >
          {message.text === "" && message.streaming === true ? (
            <Text size="14" muted={!isUser}>
              Thinking…
            </Text>
          ) : (
            <Text size="14">{message.text}</Text>
          )}
          {message.attachmentNames !== undefined && message.attachmentNames.length > 0 && (
            <Text size="11" muted={!isUser}>
              {`📎 ${message.attachmentNames.join(", ")}`}
            </Text>
          )}
          {message.error !== undefined && (
            <Text size="13" tone="error">
              {message.error}
            </Text>
          )}
        </div>
        {message.cards !== undefined && message.cards.length > 0 && (
          <Inline gap={8} wrap>
            {message.cards.map((card) => (
              <button
                key={`${card.library}/${card.id}`}
                type="button"
                onClick={() =>
                  player.play({
                    id: card.id,
                    libraryId: card.library,
                    title: card.title,
                    subtitle: card.subtitle ?? "",
                    coverUrl: card.artworkUrl ?? "",
                    streamUrl: card.streamUrl,
                  })
                }
                style={{ all: "unset", cursor: "pointer", width: 120 }}
              >
                <Stack gap={4}>
                  <CoverImage src={card.artworkUrl ?? ""} alt="" ratio="1x1" radius="8" />
                  <Text size="13">{card.title}</Text>
                  <Text size="11" muted>
                    Recommended from your Server
                  </Text>
                </Stack>
              </button>
            ))}
          </Inline>
        )}
      </Stack>
      {isUser && (
        <div style={{ flexShrink: 0, width: 28, height: 28, borderRadius: "var(--radius-pill)", background: "var(--color-accent)", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--color-bg)" }}>
          <User size={15} />
        </div>
      )}
    </Inline>
  );
}

export function LyanaScreen() {
  const moduleState = useAiModuleState();
  const chat = useLyanaChat();
  const [draft, setDraft] = useState("");
  const name = useCompanionName();
  const handsFree = useHandsFree();
  const narration = useNarration();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [pending, setPending] = useState<ChatAttachment[]>([]);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [chat.messages]);

  if (moduleState === "loading") {
    return <Page>{null}</Page>;
  }

  if (moduleState === "disabled") {
    return (
      <Page>
        <EmptyState message={`${name} isn't switched on for this server. Open Settings → Modules to turn it on.`} />
      </Page>
    );
  }

  return (
    // `fill`: the page owns the height of the shell's scroll area so the composer can sit at its bottom edge and the
    // transcript scrolls between header and composer. The shell measures the *visible* viewport, so when the keyboard
    // opens the whole shell shrinks and the composer rises with it instead of being covered by it.
    <Page fill>
      <div style={{ display: "flex", flexDirection: "column", flex: 1, minHeight: 0, gap: "var(--space-16)" }}>
        {/* "AI" in the header, never the owner's pet name: the nav label and this heading both have to work for a
            server whose companion is called something else entirely (Kevin: "people should be able to use whatever
            name they want"). The chosen name is used in the copy and the placeholder instead. */}
        <Inline justify="space-between" align="center">
          <Heading size="18">AI</Heading>
          <IconButton icon={Settings} label="Assistant settings" onClick={() => setSettingsOpen((open) => !open)} />
        </Inline>
        {settingsOpen && <CompanionSheet onClose={() => setSettingsOpen(false)} />}
        <div
          ref={logRef}
          style={{ flex: 1, minHeight: 0, display: "flex", flexDirection: "column", gap: "var(--space-16)", overflowY: "auto", overscrollBehavior: "contain" }}
        >
          {chat.messages.length === 0 ? (
            <Text muted>
              Ask {name} about anything in your library — "what should I read next", "play something from Discworld",
              or just talk.
            </Text>
          ) : (
            chat.messages.map((message) => <MessageBubble key={message.id} message={message} />)
          )}
        </div>
        <div style={{ flex: "0 0 auto" }}>
        <Composer
          value={draft}
          attachments={pending as PendingAttachment[]}
          onAttach={(files) => {
            // Read here rather than in the composer so the screen owns what is about to be sent. Size and count are
            // checked server-side too; this is the courtesy check that keeps a 40 MB video out of the request.
            void Promise.all(
              files.slice(0, 4).map(
                (file) =>
                  new Promise<ChatAttachment>((resolve, reject) => {
                    const reader = new FileReader();
                    reader.onload = () => {
                      const result = String(reader.result ?? "");
                      resolve({ name: file.name, media_type: file.type || "application/octet-stream", data: result.slice(result.indexOf(",") + 1) });
                    };
                    reader.onerror = () => reject(new Error(`Could not read ${file.name}`));
                    reader.readAsDataURL(file);
                  }),
              ),
            ).then((read) => setPending((current) => [...current, ...read].slice(0, 4)));
          }}
          onRemoveAttachment={(index) => setPending((current) => current.filter((_, position) => position !== index))}
          onChange={setDraft}
          disabled={chat.busy}
          placeholder={`Message ${name}`}
          // Only when the server can actually listen — an unusable mic is worse than none.
          onTranscribe={narration.data?.listen === true ? transcribe : undefined}
          onTranscribed={(text) => {
            if (handsFree) {
              setDraft("");
              const files = pending;
              setPending([]);
              void chat.send(text, files);
              return;
            }
            setDraft(text);
          }}
          onSend={() => {
            const text = draft;
            const files = pending;
            setDraft("");
            setPending([]);
            void chat.send(text, files);
          }}
        />
        </div>
      </div>
    </Page>
  );
}
