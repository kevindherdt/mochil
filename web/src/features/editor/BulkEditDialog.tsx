// features/editor/BulkEditDialog.tsx — every batch operation on a selection, in one place.
//
// The legacy editor had a row of buttons per operation (Set series, Set author, Renumber, Find & replace, Reset…). That
// is a lot of permanent furniture for actions used occasionally, so they live here behind one "Bulk edit…" button
// instead — but nothing was dropped: each operation below is the same one, with the same rules.
//
// Everything computed here becomes ordinary pending edits in the same changes map the cells use, so a batch is reviewed,
// saved and undone exactly like a single cell. "Reset to files" is the one exception: it asks the server to forget the
// book's edits, so it travels as the request's `reset` list instead of as a field change.
import { useState } from "react";
import { Button, Heading, Inline, Stack, Text, TextField } from "../../ui";
import {
  FIELD_LABELS,
  mergeChanges,
  renumberBooks,
  replaceInBooks,
  setAllOn,
  validateField,
  type EditorBook,
  type EditorChanges,
  type EditorField,
} from "./api/editor";

type Operation = "field" | "series" | "author" | "renumber" | "replace" | "reset";

const OPERATIONS: { id: Operation; label: string; hint: string }[] = [
  { id: "field", label: "Set a field", hint: "One value for every selected book." },
  { id: "series", label: "Set series", hint: "Same series for all, with an optional position." },
  { id: "author", label: "Set author", hint: "Join several with “ & ”." },
  { id: "renumber", label: "Renumber series", hint: "Number the selection in the order shown." },
  { id: "replace", label: "Find & replace", hint: "Swap text inside one field." },
  { id: "reset", label: "Reset to files", hint: "Forget Mochil's edits; the book's own metadata wins again." },
];

export function BulkEditDialog({
  books,
  editable,
  onCancel,
  onApply,
}: {
  books: EditorBook[];
  editable: readonly EditorField[];
  onCancel: () => void;
  onApply: (changes: EditorChanges, reset: boolean) => void;
}) {
  // An operation that writes a Calibre-owned field is not offered at all: the server would refuse it, and a control that
  // accepts work it cannot do is worse than one that is not there.
  const available = OPERATIONS.filter((entry) => {
    if (entry.id === "series") return editable.includes("series");
    if (entry.id === "author") return editable.includes("authors");
    if (entry.id === "renumber") return editable.includes("series_index");
    return true;
  });
  const [chosen, setOperation] = useState<Operation>("field");
  const operation: Operation = available.some((entry) => entry.id === chosen) ? chosen : "field";
  const [field, setField] = useState<EditorField>(editable.includes("series") ? "series" : (editable[0] ?? "series"));
  const [value, setValue] = useState("");
  const [series, setSeries] = useState("");
  const [position, setPosition] = useState("");
  const [author, setAuthor] = useState("");
  const [start, setStart] = useState("1");
  const [step, setStep] = useState("1");
  const [find, setFind] = useState("");
  const [replacement, setReplacement] = useState("");
  const [prefixOnly, setPrefixOnly] = useState(false);
  const [matchCase, setMatchCase] = useState(false);
  const [problem, setProblem] = useState<string | undefined>(undefined);

  const inputStyle = {
    width: "100%",
    boxSizing: "border-box",
    padding: "6px 8px",
    border: "1px solid var(--color-line)",
    borderRadius: "var(--radius-8)",
    background: "var(--color-bg)",
    color: "var(--color-text)",
    font: "inherit",
    fontSize: "var(--font-size-12)",
  } as const;

  function submit() {
    if (operation === "reset") {
      onApply({}, true);
      return;
    }
    if (operation === "renumber") {
      const from = Number(start);
      const by = Number(step);
      if (!Number.isFinite(from) || !Number.isFinite(by) || from <= 0 || by <= 0) {
        setProblem("Use a first number and a step above zero.");
        return;
      }
      onApply(renumberBooks(books, from, by), false);
      return;
    }
    if (operation === "replace") {
      if (find === "") {
        setProblem("Type the text to find.");
        return;
      }
      const changes = replaceInBooks(books, field, find, replacement, { prefixOnly, matchCase });
      if (Object.keys(changes).length === 0) {
        setProblem("Nothing in the selection contains that text.");
        return;
      }
      onApply(changes, false);
      return;
    }
    if (operation === "series") {
      if (series.trim() === "") {
        setProblem("Type a series name, or use Reset to files to go back to the book's own.");
        return;
      }
      let changes = setAllOn(books, "series", series);
      if (position.trim() !== "") {
        const problem = validateField("series_index", position);
        if (problem !== undefined) {
          setProblem(problem);
          return;
        }
        changes = mergeChanges(changes, setAllOn(books, "series_index", position));
      }
      onApply(changes, false);
      return;
    }
    if (operation === "author") {
      if (author.trim() === "") {
        setProblem("Type an author, or use Reset to files to go back to the book's own.");
        return;
      }
      onApply(setAllOn(books, "authors", author), false);
      return;
    }
    const issue = validateField(field, value);
    if (issue !== undefined) {
      setProblem(issue);
      return;
    }
    onApply(setAllOn(books, field, value), false);
  }

  return (
    <div
      data-bulk-dialog
      onKeyDown={(event) => {
        if (event.key === "Escape") onCancel();
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
        aria-label="Bulk edit the selected books"
        style={{
          width: "min(560px, 100%)",
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
            <Heading size="18">Bulk edit {books.length} books</Heading>
            <Text size="12" muted>
              Nothing is written yet — this stages the change so you can review it first.
            </Text>
          </Stack>

          <Stack gap={8}>
            <Text size="11" muted>
              What to do
            </Text>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))", gap: "var(--space-8)" }}>
              {available.map((entry) => {
                const active = operation === entry.id;
                return (
                  <button
                    key={entry.id}
                    type="button"
                    onClick={() => {
                      setOperation(entry.id);
                      setProblem(undefined);
                    }}
                    style={{
                      textAlign: "left",
                      padding: "8px 10px",
                      borderRadius: "var(--radius-10, 10px)",
                      border: active ? "1px solid var(--color-accent)" : "1px solid var(--color-line)",
                      background: active ? "color-mix(in srgb, var(--color-accent) 12%, transparent)" : "transparent",
                      color: "var(--color-text)",
                      font: "inherit",
                      cursor: "pointer",
                    }}
                  >
                    <div style={{ fontSize: "var(--font-size-12)" }}>{entry.label}</div>
                    <div style={{ fontSize: "10px", opacity: 0.6 }}>{entry.hint}</div>
                  </button>
                );
              })}
            </div>
          </Stack>

          {operation === "field" && (
            <Stack gap={8}>
              <label style={{ display: "block" }}>
                <Text size="11" muted>
                  Field
                </Text>
                <select value={field} onChange={(event) => setField(event.target.value as EditorField)} style={inputStyle}>
                  {editable.map((name) => (
                    <option key={name} value={name}>
                      {FIELD_LABELS[name]}
                    </option>
                  ))}
                </select>
              </label>
              <TextField
                label="New value (empty takes the field back to what the files say)"
                value={value}
                onChange={(event) => setValue(event.target.value)}
              />
            </Stack>
          )}

          {operation === "series" && (
            <Stack gap={8}>
              <TextField label="Series" value={series} onChange={(event) => setSeries(event.target.value)} />
              <TextField
                label="Position (optional — leave empty to keep each book's number)"
                value={position}
                onChange={(event) => setPosition(event.target.value)}
              />
            </Stack>
          )}

          {operation === "author" && (
            <TextField label="Author (use “ & ” between several)" value={author} onChange={(event) => setAuthor(event.target.value)} />
          )}

          {operation === "renumber" && (
            <Stack gap={8}>
              <Inline gap={12}>
                <TextField label="First number" value={start} onChange={(event) => setStart(event.target.value)} />
                <TextField label="Step" value={step} onChange={(event) => setStep(event.target.value)} />
              </Inline>
              <Text size="11" muted>
                The selected books get consecutive numbers in the order they are listed.
              </Text>
            </Stack>
          )}

          {operation === "replace" && (
            <Stack gap={8}>
              <label style={{ display: "block" }}>
                <Text size="11" muted>
                  In
                </Text>
                <select value={field} onChange={(event) => setField(event.target.value as EditorField)} style={inputStyle}>
                  {editable.map((name) => (
                    <option key={name} value={name}>
                      {FIELD_LABELS[name]}
                    </option>
                  ))}
                </select>
              </label>
              <TextField label="Find" value={find} onChange={(event) => setFind(event.target.value)} />
              <TextField
                label="Replace with (empty removes it)"
                value={replacement}
                onChange={(event) => setReplacement(event.target.value)}
              />
              <label style={{ display: "flex", gap: "var(--space-8)", alignItems: "center", fontSize: "var(--font-size-12)" }}>
                <input type="checkbox" checked={prefixOnly} onChange={(event) => setPrefixOnly(event.target.checked)} />
                Only at the start (strips a prefix like “Series - ”)
              </label>
              <label style={{ display: "flex", gap: "var(--space-8)", alignItems: "center", fontSize: "var(--font-size-12)" }}>
                <input type="checkbox" checked={matchCase} onChange={(event) => setMatchCase(event.target.checked)} />
                Match case
              </label>
            </Stack>
          )}

          {operation === "reset" && (
            <Text size="12" muted>
              This forgets every edit Mochil holds for the selected books, so their files decide the metadata again. It
              is staged like any other change and only happens when you save.
            </Text>
          )}

          {problem !== undefined && (
            <div style={{ fontSize: "var(--font-size-12)", color: "var(--color-danger, #e5484d)" }}>{problem}</div>
          )}

          <Inline gap={8}>
            <Button variant="primary" onClick={submit}>
              Stage on {books.length}
            </Button>
            <Button variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
          </Inline>
        </Stack>
      </div>
    </div>
  );
}
