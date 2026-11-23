// features/lyana/ChatsSheet.tsx — the conversations that already exist, and a way into them.
//
// The API was there all along (`GET /api/hermes/chats`, see api/chats.ts) and nothing called it, so the AI screen could
// only ever show one conversation, which started over on every reload. This lists them newest-first, opens one, starts a
// new one, and deletes one — the four things a chat list is for, and nothing else.
import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Stack, Inline, Text, Heading, Button } from "../../ui";
import { IconButton } from "../../ui";
import { deleteChat, listChats, type ChatSummary } from "./api/chats";

function when(updated: number): string {
  if (updated <= 0) return "";
  const date = new Date(updated * 1000);
  const minutes = Math.round((Date.now() - date.getTime()) / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days} d ago`;
  return date.toLocaleDateString();
}

export function ChatsSheet({
  current,
  onOpen,
  onNew,
  onClose,
}: {
  current: string;
  onOpen: (id: string) => void;
  onNew: () => void;
  onClose: () => void;
}) {
  const [chats, setChats] = useState<ChatSummary[] | null>(null);
  const [problem, setProblem] = useState("");

  const refresh = useCallback(() => {
    listChats()
      .then((list) => {
        setChats(list);
        setProblem("");
      })
      .catch(() => {
        setChats([]);
        setProblem("Could not read your chats from the server.");
      });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return (
    <div
      style={{
        border: "1px solid var(--color-line)",
        borderRadius: "var(--radius-16)",
        background: "var(--color-surface)",
        padding: "var(--space-12)",
      }}
    >
      <Stack gap={12}>
        <Inline justify="space-between" align="center">
          <Heading size="14">Chats</Heading>
          <Button
            variant="secondary"
            onClick={() => {
              onNew();
              onClose();
            }}
          >
            <Plus size={15} /> New chat
          </Button>
        </Inline>
        {problem !== "" && (
          <div style={{ color: "var(--color-danger)", fontSize: "var(--font-size-12)" }}>{problem}</div>
        )}
        {chats !== null && chats.length === 0 && problem === "" && (
          <Text size="13" muted>
            No earlier chats yet. Anything you say here is kept, so this list fills up as you use it.
          </Text>
        )}
        {chats !== null && chats.length > 0 && (
          <Stack gap={4}>
            {chats.map((chat) => (
              <Inline key={chat.id} justify="space-between" align="center" gap={8}>
                <button
                  type="button"
                  onClick={() => {
                    onOpen(chat.id);
                    onClose();
                  }}
                  style={{
                    all: "unset",
                    cursor: "pointer",
                    display: "flex",
                    flexDirection: "column",
                    gap: 2,
                    padding: "var(--space-8)",
                    borderRadius: "var(--radius-8)",
                    flex: 1,
                    minWidth: 0,
                    minHeight: "var(--chrome-touch-min)",
                    justifyContent: "center",
                    background: chat.id === current ? "var(--color-raised)" : "transparent",
                  }}
                >
                  <span
                    style={{
                      fontSize: "var(--font-size-14)",
                      color: "var(--color-text)",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {chat.title}
                  </span>
                  <span style={{ fontSize: "var(--font-size-11)", color: "var(--color-text-muted)" }}>
                    {[when(chat.updated), chat.count > 0 ? `${chat.count} messages` : ""].filter((part) => part !== "").join(" · ")}
                  </span>
                </button>
                <IconButton
                  icon={Trash2}
                  label={`Delete ${chat.title}`}
                  onClick={() => {
                    void deleteChat(chat.id)
                      .then(refresh)
                      .catch(() => setProblem("Could not delete that chat."));
                  }}
                />
              </Inline>
            ))}
          </Stack>
        )}
      </Stack>
    </div>
  );
}
