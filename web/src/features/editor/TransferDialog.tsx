// features/editor/TransferDialog.tsx — move or copy the selected books into another library.
//
// Two steps, because the first one can refuse: the server checks every book and says which would be blocked and why
// (usually a file of the same name already there). Nothing is written until the second step, and a finished transfer
// comes back with an undo token.
//
// A transfer moves files on disk, so pending edits must be saved first — a book that moves takes its saved changes with
// it, and an unsaved edit would be left pointing at a book that is no longer in this library.
import { useState } from "react";
import { Button, Heading, Inline, Stack, Text } from "../../ui";
import { applyTransfer, previewTransfer, type TransferPreview, type TransferTarget } from "./api/editor";

export function TransferDialog({
  libraryId,
  ids,
  targets,
  blocked,
  onClose,
  onDone,
}: {
  libraryId: string;
  ids: string[];
  targets: TransferTarget[];
  /** Why the transfer cannot start at all (unsaved edits, no other library), or "" when it can. */
  blocked: string;
  onClose: () => void;
  onDone: (message: string) => void;
}) {
  const [to, setTo] = useState(targets[0]?.id ?? "");
  const [mode, setMode] = useState<"move" | "copy">("copy");
  const [preview, setPreview] = useState<TransferPreview | undefined>(undefined);
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState<string | undefined>(undefined);

  async function check() {
    if (to === "") return;
    setBusy(true);
    setProblem(undefined);
    try {
      const result = await previewTransfer(libraryId, to, ids, mode);
      setPreview(result);
      if (result.blocked > 0) {
        setProblem(`${result.blocked} of ${result.ready + result.blocked} cannot go: they are listed below with the reason.`);
      }
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be checked.");
    } finally {
      setBusy(false);
    }
  }

  async function run() {
    if (to === "" || preview === undefined) return;
    setBusy(true);
    setProblem(undefined);
    try {
      const result = await applyTransfer(libraryId, to, ids, mode);
      onDone(
        `${mode === "move" ? "Moved" : "Copied"} ${ids.length} ${ids.length === 1 ? "book" : "books"} to “${preview.to}”.` +
          (result.rescan.length > 0 ? " The libraries are being rescanned." : ""),
      );
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be done.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      data-transfer-dialog
      onKeyDown={(event) => {
        if (event.key === "Escape") onClose();
      }}
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 40,
        background: "color-mix(in srgb, #000 55%, transparent)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: "var(--space-16)",
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Move or copy books"
        style={{
          width: "min(520px, 100%)",
          maxHeight: "85vh",
          overflowY: "auto",
          background: "var(--color-bg)",
          border: "1px solid var(--color-line)",
          borderRadius: "var(--radius-16)",
          padding: "var(--space-24)",
          boxShadow: "0 24px 60px rgba(0,0,0,0.35)",
        }}
      >
        <Stack gap={16}>
          <Stack gap={4}>
            <Heading size="18">
              {mode === "move" ? "Move" : "Copy"} {ids.length} {ids.length === 1 ? "book" : "books"}
            </Heading>
            <Text size="12" muted>
              Each book keeps its folder and file name in the other library.
            </Text>
          </Stack>

          {blocked !== "" ? (
            <Text size="12">{blocked}</Text>
          ) : (
            <>
              <label style={{ display: "block" }}>
                <Text size="11" muted>
                  To library
                </Text>
                <select
                  value={to}
                  onChange={(event) => {
                    setTo(event.target.value);
                    setPreview(undefined);
                  }}
                  style={{
                    width: "100%",
                    padding: "6px 8px",
                    border: "1px solid var(--color-line)",
                    borderRadius: "var(--radius-8)",
                    background: "var(--color-bg)",
                    color: "var(--color-text)",
                    font: "inherit",
                    fontSize: "var(--font-size-12)",
                  }}
                >
                  {targets.map((target) => (
                    <option key={target.id} value={target.id}>
                      {target.name}
                    </option>
                  ))}
                </select>
              </label>

              <Inline gap={8}>
                {(["copy", "move"] as const).map((option) => (
                  <button
                    key={option}
                    type="button"
                    onClick={() => {
                      setMode(option);
                      setPreview(undefined);
                    }}
                    style={{
                      padding: "6px 12px",
                      borderRadius: "var(--radius-pill)",
                      border: mode === option ? "1px solid transparent" : "1px solid var(--color-line)",
                      background: mode === option ? "var(--color-accent)" : "transparent",
                      color: mode === option ? "var(--color-accent-ink)" : "var(--color-text)",
                      font: "inherit",
                      fontSize: "var(--font-size-12)",
                      cursor: "pointer",
                    }}
                  >
                    {option === "copy" ? "Leave the original" : "Take it with me"}
                  </button>
                ))}
              </Inline>

              {preview !== undefined && (
                <Stack gap={8}>
                  <Text size="12">
                    {preview.ready} ready · {preview.blocked} blocked
                  </Text>
                  {preview.books.filter((book) => book.reason !== "").slice(0, 8).map((book) => (
                    <Text key={book.id} size="11" muted>
                      {book.title} — {book.reason}
                    </Text>
                  ))}
                </Stack>
              )}

              {problem !== undefined && (
                <div style={{ fontSize: "var(--font-size-12)", color: "var(--color-error)" }}>{problem}</div>
              )}

              <Inline gap={8} wrap>
                <Button variant="secondary" disabled={busy} onClick={() => void check()}>
                  Check first
                </Button>
                <Button
                  variant="primary"
                  disabled={busy || preview === undefined || preview.ready === 0}
                  onClick={() => void run()}
                >
                  {mode === "move" ? "Move them" : "Copy them"}
                </Button>
                <Button variant="secondary" disabled={busy} onClick={onClose}>
                  Cancel
                </Button>
              </Inline>
            </>
          )}

          {blocked !== "" && (
            <Inline gap={8}>
              <Button variant="secondary" onClick={onClose}>
                Close
              </Button>
            </Inline>
          )}
        </Stack>
      </div>
    </div>
  );
}
