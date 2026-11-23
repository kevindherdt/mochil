// features/library/screens/BookDetailScreen.tsx — fixes the real gap this slice was for: a book card opened
// nothing. `read_url` (`/api/books/{id}`'s own field) is a real, working, standalone reader page
// (`internal/web/static_http.go`'s `reader()` — confirmed live, serves `reader-readium.html` directly, not a raw
// file download), so "Read" is a genuine action today, not a placeholder.
import { useNavigate, useParams, useRouter } from "@tanstack/react-router";
import { ArrowLeft, BookOpen } from "lucide-react";
import {
  Async,
  Button,
  CoverImage,
  Heading,
  IconButton,
  Inline,
  Page,
  Stack,
  Text,
} from "../../../ui";
import { useBookDetail } from "../useMediaDetail";
import { htmlToText } from "../html";

/** A name that is also a way in: the series and the authors under a title open that series or author in the library,
 *  which is where the books themselves are. Styled as text with a dotted underline rather than as a chip — this page has
 *  no chips anywhere, and a button-shaped control here would read as an action on the book. */
function NameLink({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: 0,
        border: "none",
        background: "none",
        color: "inherit",
        font: "inherit",
        cursor: "pointer",
        textDecoration: "underline dotted",
        textUnderlineOffset: "2px",
      }}
    >
      {label}
    </button>
  );
}

export function BookDetailScreen() {
  const { libraryId, bookId } = useParams({ strict: false }) as { libraryId?: string; bookId?: string };
  const router = useRouter();
  const navigate = useNavigate();
  const query = useBookDetail(libraryId, bookId);

  /** Opens the library pre-filtered to one series or author. The library screen owns those groups already
   *  (?view=series|authors&group=…), so this is navigation, not a second implementation of the same list. */
  function openGroup(view: "series" | "authors", name: string) {
    void navigate({
      to: "/library",
      search: { lib: libraryId, view, group: name, groupName: name },
    });
  }

  return (
    <Page>
      <Stack gap={16}>
        <IconButton
          icon={ArrowLeft}
          label="Back"
          onClick={() => router.history.back()}
        />
        <Async query={query}>
          {(book) => (
            <Stack gap={16}>
              <Inline gap={16} align="start">
                <div style={{ width: 120, flexShrink: 0 }}>
                  <CoverImage
                    src={book.cover_url}
                    alt={book.title}
                    kindIcon={BookOpen}
                    radius="8"
                  />
                </div>
                <Stack gap={4}>
                  <Heading size="22">{book.title}</Heading>
                  <Text muted>
                    {book.authors.length === 0 ? (
                      "Unknown author"
                    ) : (
                      book.authors.map((author, index) => (
                        <span key={author}>
                          {index > 0 ? ", " : ""}
                          <NameLink label={author} onClick={() => openGroup("authors", author)} />
                        </span>
                      ))
                    )}
                  </Text>
                  {book.series !== undefined && (
                    <Text size="13" muted>
                      <NameLink label={book.series} onClick={() => openGroup("series", book.series!)} />
                      {book.series_index !== undefined
                        ? ` · Book ${book.series_index}`
                        : ""}
                    </Text>
                  )}
                  <Text size="13" muted>
                    {[
                      book.format?.toUpperCase(),
                      book.publisher,
                      book.published?.slice(0, 4),
                    ]
                      .filter(Boolean)
                      .join(" · ")}
                  </Text>
                </Stack>
              </Inline>
              {/* Read goes to the reader that works. The in-shell reader at /next/read/{library}/{id} is reachable and
                  improving, but it cannot turn pages reliably yet, and "Read" is the one button a book page must never
                  get wrong — the legacy reader is the same Readium build with the interaction layer already in place.
                  Switching this back is a one-line change once page turns and progress are verified. */}
              <Inline gap={8}>
                <a href={`${book.read_url}?return=${encodeURIComponent(window.location.pathname + window.location.search)}`}>
                  <Button variant="primary">Read</Button>
                </a>
              </Inline>
              {book.description !== undefined && (
                <Text size="14">{htmlToText(book.description)}</Text>
              )}
            </Stack>
          )}
        </Async>
      </Stack>
    </Page>
  );
}
