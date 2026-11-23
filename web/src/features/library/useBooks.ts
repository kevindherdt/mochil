// features/library/useBooks.ts — the hook for a shelf's book grid, paged with the server's own cursor.
//
// Keyed on library + drill-down filter so switching the PillRow selection (or opening one series/author)
// re-fetches, while React Query keeps the previous view's cache warm for a quick switch back. Paging is a
// useInfiniteQuery rather than a growing array in the screen: the cursor belongs to the query, so a filter change
// starts a new chain instead of appending page 2 of the old view onto the new one — the failure mode a manual
// "load more" array invites.
import { useInfiniteQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchBooksPage, type BookFilter, type BookInfo } from "./api/books";

/** 120 is the window the server serves comfortably (and what the legacy client asked for); one page is ~8 rows of
 *  a phone grid, so the first paint is complete without fetching a 5430-book shelf up front. */
export const BOOK_PAGE_SIZE = 120;

export interface BooksState {
  query: AsyncQueryLike<BookInfo[]>;
  hasMore: boolean;
  loadMore: () => void;
  isLoadingMore: boolean;
  /** Size of the whole filtered shelf (X-Total-Count), or undefined before the first response. */
  total: number | undefined;
}

export function useBooks(libraryId: string | undefined, filter: BookFilter = {}): BooksState {
  const query = useInfiniteQuery({
    queryKey: [
      "library",
      "books",
      libraryId,
      filter.series ?? "",
      filter.author ?? "",
      filter.q ?? "",
      filter.status ?? "",
    ],
    queryFn: ({ pageParam }) => fetchBooksPage(libraryId!, filter, pageParam === null ? undefined : pageParam, BOOK_PAGE_SIZE),
    initialPageParam: null as string | null,
    // The server decides when the shelf ends: a null/empty cursor is the last page. Never "page length < size",
    // which would drop the final window whenever the shelf size happens to be an exact multiple of the page size.
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled: libraryId !== undefined,
    staleTime: 30_000,
  });

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const firstPage = query.data?.pages[0];

  return {
    query: {
      data: query.data === undefined ? undefined : items,
      isLoading: query.isLoading,
      isError: query.isError,
      error: query.error,
      refetch: () => void query.refetch(),
    },
    hasMore: query.hasNextPage,
    loadMore: () => void query.fetchNextPage(),
    isLoadingMore: query.isFetchingNextPage,
    total: firstPage?.total ?? undefined,
  };
}
