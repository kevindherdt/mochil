// features/library/CollectionsView.tsx — the "Collections" browse mode, and the books inside one.
//
// Collections were already in the API and in the legacy app (a shelf of hand-made and smart lists) but had no
// rewrite UI: the Library screen could browse All books / Series / Authors and nothing else.
//
// Drawn with the same GroupCard as Series/Authors, stacked, because a collection is exactly that shape — a named
// group of books with a count and a cover. A smart collection is marked rather than styled differently: it behaves
// identically when you open it, and only the "why is this book here" answer differs.
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Async, Button, GroupCard, Inline, MediaGrid, MediaList, Text } from "../../ui";
import { Layers } from "lucide-react";
import { fetchCollectionBooks, fetchCollections } from "./api/collections";
import { displayTitle } from "./titles";

export function CollectionsList({
  libraryId,
  layout,
  onOpen,
}: {
  libraryId: string | undefined;
  layout: "grid" | "list";
  onOpen: (id: string, name: string) => void;
}) {
  const query = useQuery({
    queryKey: ["library", "collections", libraryId],
    queryFn: () => fetchCollections(libraryId as string),
    enabled: libraryId !== undefined,
    staleTime: 30_000,
  });

  return (
    <Async query={query} empty={(collections) => collections.length === 0} emptyMessage="No collections in this shelf yet.">
      {(collections) =>
        layout === "list" ? (
          <MediaList
            items={collections.map((collection) => ({
              id: collection.id,
              title: collection.name,
              subtitle: `${collection.count === 1 ? "1 book" : `${collection.count} books`}${collection.smart ? " · smart" : ""}`,
              coverUrl: collection.coverUrl,
              onClick: () => onOpen(collection.id, collection.name),
            }))}
          />
        ) : (
          <MediaGrid>
            {collections.map((collection) => (
              <GroupCard
                key={collection.id}
                name={collection.name}
                count={collection.count}
                countLabel={`${collection.count === 1 ? "1 book" : `${collection.count} books`}${collection.smart ? " · smart" : ""}`}
                coverUrl={collection.coverUrl}
                kindIcon={Layers}
                stack
                onClick={() => onOpen(collection.id, collection.name)}
              />
            ))}
          </MediaGrid>
        )
      }
    </Async>
  );
}

export function CollectionBooks({
  collectionId,
  layout,
  name,
  onBack,
}: {
  collectionId: string;
  layout: "grid" | "list";
  name: string;
  onBack: () => void;
}) {
  const navigate = useNavigate();
  const query = useQuery({
    queryKey: ["library", "collection-books", collectionId],
    queryFn: () => fetchCollectionBooks(collectionId),
    staleTime: 30_000,
  });

  const open = (libraryId: string, bookId: string) =>
    void navigate({ to: "/library/book/$libraryId/$bookId", params: { libraryId, bookId } });

  return (
    <>
      <Inline gap={8} align="center">
        <Button variant="secondary" onClick={onBack}>
          ← All collections
        </Button>
        <Text size="13" muted>
          {name}
        </Text>
      </Inline>
      <Async query={query} empty={(result) => result.books.length === 0} emptyMessage="This collection is empty.">
        {(result) =>
          layout === "list" ? (
            <MediaList
              items={result.books.map((book) => ({
                id: book.id,
                title: displayTitle(book),
                subtitle: book.authors.join(", ") || "Unknown author",
                coverUrl: book.cover_url ?? "",
                progress: book.reading_progress?.finished ? undefined : book.reading_progress?.fraction,
                finished: book.reading_progress?.finished,
                onClick: () => open(book.library_id, book.id),
              }))}
            />
          ) : (
            <MediaGrid>
              {result.books.map((book) => (
                <GroupCard
                  key={book.id}
                  name={displayTitle(book)}
                  count={1}
                  countLabel={book.authors.join(", ") || "Unknown author"}
                  coverUrl={book.cover_url ?? ""}
                  kindIcon={undefined}
                  onClick={() => open(book.library_id, book.id)}
                />
              ))}
            </MediaGrid>
          )
        }
      </Async>
    </>
  );
}
