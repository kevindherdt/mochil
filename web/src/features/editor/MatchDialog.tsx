// features/editor/MatchDialog.tsx — the candidates behind a match, and a second search with better words.
//
// Kevin, on a row whose match read "Lovelace, Volume 07 · 16%": "its not smart enough to see its wrong or to look at
// match scores of different possibilities? That said where can I see this?"
//
// The finding was never hidden — it was just never shown. The editor carried one candidate per book, so a 16% match
// looked exactly like a certainty with a number in front of it, and the other seven candidates Hardcover returned were
// thrown away before they reached the screen. This dialog is where they live: the ranked list, their scores coloured on
// the same scale as the table, and the two words that decide whether the search works at all — the title and author,
// editable, because "‑EIGHTY‑SIX - Volume 07" is not a title anyone would search for.
import * as Dialog from "@radix-ui/react-dialog";
import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { Button, Heading, Inline, Stack, Text, TextField } from "../../ui";
import { searchMatches, type EditorBook, type EditorMatch } from "./api/editor";
import { scoreLabel, scoreTone, toneInk } from "./tone";

export function MatchDialog({
  libraryId,
  book,
  onUse,
  onClose,
}: {
  libraryId: string;
  book: EditorBook;
  /** Take this candidate: the caller stages the same fields a bulk Replace would. */
  onUse: (candidate: EditorMatch) => void;
  onClose: () => void;
}) {
  // The book's own words are the starting point, not the answer: the point of the dialog is that they can be changed.
  const [title, setTitle] = useState(book.title);
  const [author, setAuthor] = useState(book.authors[0] ?? "");
  const [candidates, setCandidates] = useState<EditorMatch[]>([]);
  const [searched, setSearched] = useState<string | undefined>(undefined);
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState<string | undefined>(undefined);

  async function search() {
    setBusy(true);
    setProblem(undefined);
    try {
      const found = await searchMatches(libraryId, book.id, { title, author });
      setCandidates(found.candidates);
      setSearched(`${found.title}${found.authors.length > 0 ? ` · ${found.authors[0]}` : ""}`);
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "Hardcover could not be asked.");
    } finally {
      setBusy(false);
    }
  }

  // Search on open: the stored finding is one candidate for a book nobody has decided about, and seeing the rest is
  // the whole reason this was opened.
  //
  // Deliberately not `search()`: that one sets the busy flag before it awaits, and a state write made during an effect
  // is a render loop waiting to happen. The opening search only writes when its answer arrives.
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const found = await searchMatches(libraryId, book.id, { title: book.title, author: book.authors[0] ?? "" });
        if (cancelled) return;
        setCandidates(found.candidates);
        setSearched(`${found.title}${found.authors.length > 0 ? ` · ${found.authors[0]}` : ""}`);
      } catch (error) {
        if (!cancelled) setProblem(error instanceof Error ? error.message : "Hardcover could not be asked.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [libraryId, book.id, book.title, book.authors]);

  const best = candidates[0]?.score;

  // Radix, not a fixed div: an overlay has to keep focus inside itself, close on Escape and stop the page scrolling
  // underneath, and a hand-rolled one gets two of those three wrong (this one did — Tab walked out into the table and
  // the editor's keys kept driving it while the dialog was open). Same shell as the three sibling dialogs.
  return (
    <Dialog.Root open onOpenChange={(next) => (next ? undefined : onClose())}>
      <Dialog.Overlay style={{ position: "fixed", inset: 0, padding: "var(--space-16)", background: "color-mix(in srgb, #000 55%, transparent)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 40 }}>
        <Dialog.Content
          aria-label={`Matches for ${book.title}`}
          style={{ width: "min(680px, 92vw)", maxHeight: "84vh", overflow: "auto", background: "var(--color-bg)", border: "1px solid var(--color-line)", borderRadius: "var(--radius-16)", padding: "var(--space-24)", boxShadow: "0 24px 60px color-mix(in srgb, #000 35%, transparent)" }}
        >
        <Stack gap={12}>
          <Stack gap={4}>
            <Heading size="16">Matches for {book.title}</Heading>
            <Text size="11" muted>
              {book.series === "" ? "No series in the metadata" : `${book.series}${book.seriesIndex === undefined ? "" : ` #${book.seriesIndex}`}`}
              {" · "}the search words decide what comes back, so change them if this is the wrong book.
            </Text>
          </Stack>

          <Inline gap={8} align="end" wrap>
            <div style={{ flex: "2 1 260px", minWidth: 0 }}>
              <TextField label="Search for" value={title} onChange={(event) => setTitle(event.target.value)} onKeyDown={(event) => event.key === "Enter" && void search()} />
            </div>
            <div style={{ flex: "1 1 160px", minWidth: 0 }}>
              <TextField label="Author" value={author} onChange={(event) => setAuthor(event.target.value)} onKeyDown={(event) => event.key === "Enter" && void search()} />
            </div>
            <Button variant="primary" disabled={busy} onClick={() => void search()}>
              <Search size={14} aria-hidden="true" /> {busy ? "Searching…" : "Search"}
            </Button>
          </Inline>

          {problem !== undefined && <div style={{ fontSize: "var(--font-size-12)", color: "var(--color-error)" }}>{problem}</div>}

          {searched !== undefined && (
            <Text size="11" muted>
              {candidates.length} result{candidates.length === 1 ? "" : "s"} for “{searched}”
              {best !== undefined && best < 0.5 ? " — nothing above 50%, so this book is probably spelled differently or is not on Hardcover" : ""}
            </Text>
          )}

          <Stack gap={8}>
            {candidates.map((candidate, index) => (
              <div
                key={`${candidate.providerId}-${index}`}
                style={{ border: "1px solid var(--color-line)", borderRadius: "var(--radius-8)", padding: "var(--space-8)" }}
              >
                <Inline gap={8} align="center" wrap>
                  <span style={{ fontSize: "var(--font-size-11)", fontWeight: 700, color: toneInk(scoreTone(candidate.score)), minWidth: 44 }}>
                    {scoreLabel(candidate.score)}
                  </span>
                  <Stack gap={4}>
                    <Text size="13">{candidate.title}</Text>
                    <Text size="11" muted>
                      {candidate.authors.join(" & ")}
                      {candidate.year > 0 ? ` · ${candidate.year}` : ""}
                      {candidate.series !== "" ? ` · ${candidate.series}${candidate.seriesIndex !== undefined ? ` #${candidate.seriesIndex}` : ""}` : ""}
                    </Text>
                    {/* The blurb is why the search is per book: two editions of the same title are told apart by what
                        they are about, not by their names. Clamped, because this is a list and not a reading pane. */}
                    {candidate.overview !== undefined && candidate.overview !== "" && (
                      <Text size="11" muted>
                        {candidate.overview.length > 180 ? `${candidate.overview.slice(0, 177)}…` : candidate.overview}
                      </Text>
                    )}
                  </Stack>
                  <div style={{ flex: 1 }} />
                  <Button variant="secondary" onClick={() => onUse(candidate)} title="Stage this choice as pending changes: the fields it disagrees with, reviewed before anything is written.">
                    Use this
                  </Button>
                </Inline>
              </div>
            ))}
            {searched !== undefined && candidates.length === 0 && !busy && (
              <Text size="12" muted>
                Nothing came back. Try the series name and the volume number, or just the author.
              </Text>
            )}
          </Stack>

          <Inline justify="space-between" align="center">
            <Text size="11" muted>
              Choosing a match stages the difference; nothing is written until you save.
            </Text>
            <Button variant="secondary" onClick={onClose}>
              Close
            </Button>
          </Inline>
        </Stack>
        </Dialog.Content>
      </Dialog.Overlay>
    </Dialog.Root>
  );
}
