// features/editor/ReviewPanel.tsx — what "Review changes" shows before anything is written.
//
// The first version was one wrapped line per change: title, field, value → value. After an ISBN search plus a Hardcover
// match that produced a wall of paragraph-length descriptions with no grouping and no way to tell which value had been
// added to a comma-separated list — Kevin: "its very hard to look at and see what it matched to and the difference".
//
// So this panel does three things the flat list could not:
//   - one block per book, so the changes to a book are read together and the count is per book, not per field;
//   - the book's Hardcover match is named at the top of its block ("matched to <title> · 92%"), because "hardcover:974285"
//     is an id, not an answer to "what did it match to";
//   - a diff that shows *what changed*: lists are compared as items, so an added isbn:/hardcover: token is the only thing
//     marked, and long prose is cut to a couple of lines with the rest one click away.
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { Button, Heading, Inline, Stack, Text } from "../../ui";
import { FIELD_LABELS, type EditorBook, type EditorChange, type EditorField } from "./api/editor";
import { scoreTone, toneColor } from "./tone";

/**
 * How a stored value reads in the panel. Go's zero time travels as "0101-01-01T00:00:00+00:00", which is not a year
 * anyone set — it means "not set", and printing it beside the new value made the change look like a repair of garbage.
 */
function display(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") return "(not set)";
  // Go's zero time is year 1 or 101 depending on the source; no book was published then.
  const iso = /^(\d{4})-\d{2}-\d{2}T/.exec(trimmed);
  if (iso !== null && Number(iso[1]) < 1800) return "(not set)";
  return trimmed;
}

/** A comma-separated value (identifiers, tags) as its items — the unit a person reads a change in. */
function items(value: string): string[] {
  return value
    .split(/[,;]/)
    .map((part) => part.trim())
    .filter((part) => part !== "");
}

function ListDiff({ from, to }: { from: string; to: string }) {
  const before = new Set(items(from));
  const after = items(to);
  return (
    <Inline gap={4} wrap>
      {after.map((item) => {
        const added = !before.has(item);
        return (
          <span
            key={item}
            style={{
              fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
              fontSize: "var(--font-size-11)",
              padding: "1px 6px",
              borderRadius: "var(--radius-6)",
              background: added ? "color-mix(in srgb, var(--color-accent) 18%, transparent)" : "var(--color-raised)",
              color: added ? "var(--color-accent)" : "var(--color-muted)",
            }}
            title={added ? "Added by this change" : "Already there"}
          >
            {item}
          </span>
        );
      })}
    </Inline>
  );
}

function Prose({ value, limit = 220 }: { value: string; limit?: number }) {
  const [whole, setWhole] = useState(false);
  const long = value.length > limit;
  const shown = whole || !long ? value : `${value.slice(0, limit).trimEnd()}…`;
  return (
    <Stack gap={4}>
      <Text size="12">{shown}</Text>
      {long && (
        <button
          type="button"
          onClick={() => setWhole((current) => !current)}
          style={{
            all: "unset",
            cursor: "pointer",
            color: "var(--color-accent)",
            fontSize: "var(--font-size-11)",
            display: "inline-flex",
            alignItems: "center",
            gap: "var(--space-4)",
          }}
        >
          {whole ? "Show less" : `Show all ${value.length.toLocaleString()} characters`}
        </button>
      )}
    </Stack>
  );
}

const LIST_FIELDS = new Set(["identifiers", "tags"]);

function ChangeRow({ change }: { change: EditorChange }) {
  const label = FIELD_LABELS[change.field as EditorField] ?? change.field;
  const from = change.from.trim();
  const to = change.to.trim();
  return (
    <div style={{ display: "grid", gridTemplateColumns: "96px minmax(0, 1fr)", gap: "var(--space-8)", alignItems: "start" }}>
      <Text size="11" muted>
        {label}
      </Text>
      <div>
        {LIST_FIELDS.has(change.field) ? (
          <ListDiff from={from} to={to} />
        ) : change.field === "description" ? (
          <Stack gap={4}>
            {from === "" ? (
              <Text size="11" muted>
                was empty
              </Text>
            ) : (
              <Text size="11" muted>
                replaced {from.length.toLocaleString()} characters
              </Text>
            )}
            <Prose value={to} />
          </Stack>
        ) : (
          <Inline gap={4} wrap>
            <Text size="12" muted>
              {display(from)}
            </Text>
            <Text size="11" muted>
              →
            </Text>
            <Text size="12">{display(to)}</Text>
          </Inline>
        )}
      </div>
    </div>
  );
}

export function ReviewPanel({
  changes,
  books,
  busy,
  onApply,
  onCancel,
  onApplyBook,
  onDiscardBook,
  onDiscardAll,
}: {
  changes: EditorChange[];
  books: EditorBook[];
  busy: boolean;
  onApply: () => void;
  onCancel: () => void;
  /** Write just this book. The rest of the batch stays staged. */
  onApplyBook: (bookId: string) => void;
  /** Drop just this book's staged changes. */
  onDiscardBook: (bookId: string) => void;
  /** Drop every staged change: a batch of 38 is often "none of these". */
  onDiscardAll: () => void;
}) {
  const byBook = new Map<string, { title: string; changes: EditorChange[] }>();
  for (const change of changes) {
    const entry = byBook.get(change.book) ?? { title: change.title, changes: [] };
    entry.changes.push(change);
    byBook.set(change.book, entry);
  }
  const order = [...byBook.keys()];
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  if (changes.length === 0) {
    return (
      <div style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-12)", padding: "var(--space-16)" }}>
        <Heading size="16">Nothing would change</Heading>
      </div>
    );
  }

  return (
    <div style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-12)", padding: "var(--space-16)", background: "var(--color-surface)" }}>
      <Stack gap={12}>
        <Inline justify="space-between" align="center" wrap gap={8}>
          <Stack gap={4}>
            <Heading size="16">
              {changes.length} change{changes.length === 1 ? "" : "s"} in {order.length} book{order.length === 1 ? "" : "s"}
            </Heading>
            <Text size="11" muted>
              Nothing is written until you press Apply.
            </Text>
          </Stack>
          <Inline gap={8}>
            <Button variant="primary" disabled={busy} onClick={onApply}>
              Apply {changes.length} change{changes.length === 1 ? "" : "s"}
            </Button>
            <Button variant="secondary" disabled={busy} onClick={onDiscardAll} title="Drop every staged change. Nothing is written, and the books are left out of the next staging too.">
              Discard all
            </Button>
            <Button variant="secondary" disabled={busy} onClick={onCancel}>
              Cancel
            </Button>
          </Inline>
        </Inline>

        <div style={{ maxHeight: 420, overflowY: "auto", paddingRight: "var(--space-4)" }}>
          <Stack gap={12}>
            {order.map((id) => {
              const entry = byBook.get(id);
              if (entry === undefined) return null;
              const book = books.find((candidate) => candidate.id === id);
              const open = !collapsed.has(id);
              return (
                <div key={id} style={{ borderTop: "1px solid var(--color-line)", paddingTop: "var(--space-8)" }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "var(--space-8)", flexWrap: "wrap" }}>
                  <button
                    type="button"
                    aria-expanded={open}
                    onClick={() =>
                      setCollapsed((current) => {
                        const next = new Set(current);
                        if (next.has(id)) next.delete(id);
                        else next.add(id);
                        return next;
                      })
                    }
                    style={{ all: "unset", cursor: "pointer", display: "flex", alignItems: "center", gap: "var(--space-8)", flex: "1 1 auto", minWidth: 0 }}
                  >
                    {open ? <ChevronDown size={14} aria-hidden="true" /> : <ChevronRight size={14} aria-hidden="true" />}
                    <Text size="13">{entry.title}</Text>
                    <Text size="11" muted>
                      {entry.changes.length} field{entry.changes.length === 1 ? "" : "s"}
                    </Text>
                  </button>
                  {/* Per book, because a batch of 38 is often "these 30 are right and these 8 need a look". Applying one
                      book writes it and leaves the rest staged; discarding one drops it without writing anything. */}
                  <span style={{ marginLeft: "auto", display: "inline-flex", gap: "var(--space-8)" }}>
                    <Button variant="secondary" disabled={busy} onClick={() => onApplyBook(id)}>
                      Apply this book
                    </Button>
                    <Button variant="secondary" disabled={busy} onClick={() => onDiscardBook(id)}>
                      Discard
                    </Button>
                  </span>
                  </div>

                  {/* What the match search chose for this book. Without it the only clue is an id inside the identifiers
                      string, which says nothing about which edition it picked. The series is named here too, because
                      it is the field "Replace" is most often used for and the diff alone does not say where it came
                      from. */}
                  {book?.match !== undefined && (
                    <Text size="11" muted>
                      Hardcover: {book.match.title}
                      {book.match.year !== undefined && book.match.year > 0 ? ` (${book.match.year})` : ""} ·{" "}
                      <span style={{ color: toneColor(scoreTone(book.match.score)), fontWeight: 600 }}>
                        {Math.round(book.match.score * 100)}%
                      </span>
                      {book.match.series !== "" ? ` · ${book.match.series}${book.match.seriesIndex !== undefined ? ` #${book.match.seriesIndex}` : ""}` : ""}
                      {book.matchClear ? "" : " — not confident"}
                    </Text>
                  )}

                  {open && (
                    <Stack gap={8}>
                      {entry.changes.map((change, index) => (
                        <ChangeRow key={`${change.field}-${index}`} change={change} />
                      ))}
                    </Stack>
                  )}
                </div>
              );
            })}
          </Stack>
        </div>


      </Stack>
    </div>
  );
}
