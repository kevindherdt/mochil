// features/library/VideoMatchSheet.tsx — the pencil behind a film or a series: fix its TMDB match by hand.
//
// Kevin: "for the tmdb matching I dont want a big button like before it should be out of the way somewhere or being
// able to click on the series/movie/ etc and have a edit button like a pencil or something where I can do these
// things so its clean and hidden away". So there is no "Match library" button anywhere in the app. A pencil sits
// next to the title of the film or series you are already looking at, and this opens under it: search, pick, done.
// The library-wide matcher still exists on the server for scripts; the UI never offers it.
//
// The search is a TanStack query rather than an effect: `query` starts as the scanner's own title, so there is no
// "retype the name into an empty box" step, and editing the text re-queries. A series is searched as TV and a film
// as a film — otherwise TMDB answers with films that happen to share the show's name.
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Stack, Inline, Text, TextField, Button, CoverImage } from "../../ui";
import { messageFromError } from "../settings/api/accounts";
import { candidateToOverride, readItemOverride, readShowOverride, searchMatches, type VideoOverride } from "./api/videoMatch";

export interface VideoMatchSheetProps {
  libraryId: string;
  /** The item id for a film, undefined for a series. */
  itemId: string | undefined;
  /** The series name, when matching a series. */
  show: string | undefined;
  initialQuery: string;
  onSave: (override: VideoOverride) => Promise<void>;
  onClear: () => Promise<void>;
  onClose: () => void;
}

export function VideoMatchSheet({ libraryId, itemId, show, initialQuery, onSave, onClear, onClose }: VideoMatchSheetProps) {
  const [query, setQuery] = useState(initialQuery);
  const [submitted, setSubmitted] = useState(initialQuery);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const series = itemId === undefined;

  // What it points at right now — fetched here rather than passed in, so a screen only has to say *what* to match.
  const existing = useQuery({
    queryKey: ["video-match-current", libraryId, show ?? itemId ?? ""],
    queryFn: () => (itemId === undefined ? readShowOverride(libraryId, show ?? "") : readItemOverride(libraryId, itemId)),
    staleTime: 0,
  });
  const current: VideoOverride | undefined = existing.data;

  const found = useQuery({
    queryKey: ["video-match", libraryId, show ?? itemId ?? "", series, submitted],
    queryFn: () => searchMatches(libraryId, submitted, series),
    enabled: submitted.trim() !== "",
    staleTime: 60_000,
  });

  async function apply(action: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      onClose();
    } catch (caught) {
      setError(messageFromError(caught));
    } finally {
      setBusy(false);
    }
  }

  const results = found.data;
  const searching = found.isFetching || busy;
  const failure = error ?? (found.error === null || found.error === undefined ? null : messageFromError(found.error));

  return (
    <Stack gap={12}>
      <Inline justify="space-between" align="center">
        <Text size="13">{series ? "Match series to TMDB" : "Match film to TMDB"}</Text>
        <Button variant="secondary" onClick={onClose}>
          Done
        </Button>
      </Inline>

      {current !== undefined && (
        <Text size="11" muted>
          Now pointing at {current.title}
          {current.year === undefined ? "" : ` (${current.year})`}. Choosing another replaces it.
        </Text>
      )}

      <Inline gap={8} align="end">
        <div style={{ flex: 1, minWidth: 0 }}>
          <TextField
            label="Title"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") setSubmitted(query.trim());
            }}
          />
        </div>
        <Button variant="secondary" disabled={searching || query.trim() === ""} onClick={() => setSubmitted(query.trim())}>
          Search
        </Button>
      </Inline>

      {searching && (
        <Text size="11" muted>
          Searching TMDB…
        </Text>
      )}
      {failure !== null && (
        <Text size="13" tone="error">
          {failure}
        </Text>
      )}

      {results !== undefined && results.length === 0 && !searching && (
        <Text size="12" muted>
          Nothing on TMDB for “{submitted}”. Try the original title, or add the year.
        </Text>
      )}

      {results !== undefined && results.length > 0 && (
        <Stack gap={4}>
          {results.map((candidate) => (
            <button
              key={candidate.providerId}
              type="button"
              disabled={busy}
              onClick={() => void apply(() => onSave(candidateToOverride(candidate)))}
              style={{
                all: "unset",
                cursor: "pointer",
                display: "flex",
                gap: "var(--space-12)",
                alignItems: "center",
                padding: "var(--space-8) 0",
                borderBottom: "1px solid var(--color-line)",
                width: "100%",
                boxSizing: "border-box",
              }}
            >
              <div style={{ width: 40, flexShrink: 0 }}>
                <CoverImage src={candidate.posterUrl} alt="" ratio="2x3" radius="8" />
              </div>
              <Stack gap={4}>
                <Text size="14">
                  {candidate.title}
                  {candidate.year === undefined ? "" : ` · ${candidate.year}`}
                </Text>
                {candidate.overview !== "" && (
                  <Text size="11" muted>
                    {candidate.overview.length > 90 ? `${candidate.overview.slice(0, 88)}…` : candidate.overview}
                  </Text>
                )}
              </Stack>
            </button>
          ))}
        </Stack>
      )}

      {current !== undefined && (
        <div>
          <Button variant="secondary" disabled={busy} onClick={() => void apply(onClear)}>
            Remove this match
          </Button>
        </div>
      )}
    </Stack>
  );
}
