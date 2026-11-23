// features/settings/screens/MetadataReviewScreen.tsx — Settings → Metadata → Review.
//
// Phase 2.8's "metadata review" was the last thing in that line with a real backend and no UI: the server already
// inspects every book and reports what is wrong (`/api/metadata/issues` → `metadata.Report`, internal/metadata/
// inspection.go). Before this, the only way to see that a shelf had 300 books with no author was the legacy page.
//
// What it deliberately does not do: apply fixes itself. The per-book editor lives at
// `/api/metadata/editor/<library>/<book>` and belongs on the book's own screen; this is a triage list — what is
// wrong, how bad, and a way to get to the book. Offering a second edit surface here would duplicate that screen and
// drift from it.
//
// The endpoint is per-library and answers 409 while a shelf's embedded metadata is still hydrating, with a
// Retry-After header. That is a real state, not an error to shout about: it renders as "still loading, try again".
import { useQuery } from "@tanstack/react-query";
import { Link, useSearch } from "@tanstack/react-router";
import { z } from "zod";
import { Page, Stack, Inline, Heading, Text, Card, Async, Button } from "../../../ui";
import { request } from "../../../lib/http";
import { HttpError } from "../../../lib/http";
import { useLibraries } from "../../library/useLibraries";

const issueSchema = z.looseObject({
  book_id: z.string(),
  title: z.string().optional(),
  authors: z.array(z.string()).optional(),
  format: z.string().optional(),
  code: z.string(),
  reason: z.string(),
  confidence: z.string().optional(),
});

const reportSchema = z.looseObject({
  books_inspected: z.number(),
  issue_count: z.number(),
  counts: z.record(z.string(), z.number()).optional(),
  // The server marshals a nil slice as `null` when a shelf has no issues at all (Go's json.Marshal), which is not
  // the same thing as the field being absent — a bare z.array() rejects it and turned "no issues" into an error.
  issues: z
    .array(issueSchema)
    .nullable()
    .transform((value) => value ?? []),
});

type Issue = z.infer<typeof issueSchema>;

export function MetadataReviewScreen() {
  const search = useSearch({ strict: false }) as { lib?: string };
  const libraries = useLibraries();
  const books = (libraries.data ?? []).filter(
    (library) => library.kind === "books" || library.kind === "manga" || library.kind === "comics" || library.kind === "audiobooks",
  );
  const libraryId = search.lib ?? books[0]?.id;

  const query = useQuery({
    queryKey: ["metadata", "issues", libraryId],
    queryFn: async () => {
      try {
        const raw = await request<unknown>(`/api/metadata/issues?library=${encodeURIComponent(libraryId as string)}&limit=500`);
        return { kind: "report" as const, report: reportSchema.parse(raw) };
      } catch (error) {
        // 409 is the server saying "this shelf is still loading its embedded metadata" — a wait, not a failure.
        if (error instanceof HttpError && error.status === 409) return { kind: "hydrating" as const };
        throw error;
      }
    },
    enabled: libraryId !== undefined,
    staleTime: 30_000,
  });

  return (
    <Page>
      <Stack gap={16}>
        <Heading level={1}>Metadata review</Heading>
        <Text size="13" muted>
          What the catalogue is missing — titles, authors, series gaps and lookalike editions — as the server sees it.
          Open a book to fix it; this list only triages.
        </Text>

        {books.length > 1 && (
          <Inline gap={8} align="center" wrap>
            {books.map((library) => (
              <Link key={library.id} to="/settings/metadata/review" search={{ lib: library.id }} style={{ textDecoration: "none" }}>
                <Button variant={library.id === libraryId ? "primary" : "secondary"}>{library.name}</Button>
              </Link>
            ))}
          </Inline>
        )}

        <Async query={query}>
          {(result) => {
            if (result.kind === "hydrating") {
              return (
                <Card>
                  <Stack gap={8}>
                    <Text size="14">This shelf is still reading its embedded metadata.</Text>
                    <Text size="13" muted>
                      The inspection will be accurate once it finishes — try again in a minute. Nothing is wrong with
                      the books.
                    </Text>
                  </Stack>
                </Card>
              );
            }
            const report = result.report;
            return (
              <Stack gap={16}>
                <Text size="13" muted>
                  {report.issue_count === 0
                    ? `No issues found in ${report.books_inspected} books.`
                    : `${report.issue_count} issues across ${report.books_inspected} books.`}
                </Text>
                {report.issue_count > 0 && (
                  <Stack gap={8}>
                    {report.issues.map((issue, index) => (
                      <IssueRow key={`${issue.book_id}-${issue.code}-${index}`} issue={issue} libraryId={libraryId as string} />
                    ))}
                  </Stack>
                )}
              </Stack>
            );
          }}
        </Async>
      </Stack>
    </Page>
  );
}

function IssueRow({ issue, libraryId }: { issue: Issue; libraryId: string }) {
  const tone = issue.confidence === "high" ? "error" : undefined;
  return (
    <Link
      to="/library/book/$libraryId/$bookId"
      params={{ libraryId, bookId: issue.book_id }}
      style={{ textDecoration: "none", color: "inherit" }}
    >
      <Card>
        <Stack gap={4}>
          <Inline justify="space-between" align="center" gap={8}>
            <Text size="14">{issue.title !== undefined && issue.title !== "" ? issue.title : issue.book_id}</Text>
            <Text size="11" {...(tone === undefined ? {} : { tone })}>
              {issue.confidence ?? ""}
            </Text>
          </Inline>
          <Text size="13" muted>
            {issue.reason}
          </Text>
          <Text size="11" muted>
            {[issue.code, issue.format?.toUpperCase(), (issue.authors ?? []).join(", ")].filter((part) => part !== undefined && part !== "").join(" · ")}
          </Text>
        </Stack>
      </Card>
    </Link>
  );
}
