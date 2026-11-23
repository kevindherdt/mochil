// features/library/useShelfMedia.ts — one hook for the shelves that are not /api/books: video and audio. It picks
// the endpoint from the library kind, pages with `offset` (these endpoints answer limit/offset + X-Total-Count, not
// the catalogue's cursor), and normalises to ShelfItem so the screen has one shape to render.
//
// The dispatch lives here rather than in the screen because it is server knowledge: which store a kind lives in is
// a fact about this server, not about the phone grid that draws it.
import { useInfiniteQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchAudioItems, fetchShows, fetchVideoItems, type ShelfItem } from "./api/media";
import type { ShelfSource } from "./kinds";

export const MEDIA_PAGE_SIZE = 120;

export interface ShelfMediaState {
  query: AsyncQueryLike<ShelfItem[]>;
  hasMore: boolean;
  loadMore: () => void;
  isLoadingMore: boolean;
}

/** Audio-only ordering/filtering. Both are the server's parameters, so they are part of the query key — changing
 *  "unplayed" must refetch page 1, not filter the pages already loaded in the browser. */
export interface ShelfAudioFilter {
  sort?: "title" | "recent" | "oldest" | undefined;
  played?: "played" | "unplayed" | "continue" | undefined;
  /** Drill into one podcast show. */
  feed?: string | undefined;
}

export function useShelfMedia(
  libraryId: string | undefined,
  source: Exclude<ShelfSource, "catalogue">,
  q = "",
  audio: ShelfAudioFilter = {},
): ShelfMediaState {
  const sort = audio.sort ?? "title";
  const played = audio.played;
  const feed = audio.feed;
  const query = useInfiniteQuery({
    queryKey: ["library", "media", libraryId, source, q, sort, played ?? "", feed ?? ""],
    queryFn: ({ pageParam }) => {
      const options = { q, offset: pageParam, limit: MEDIA_PAGE_SIZE, sort, played, feed };
      if (source === "audio") return fetchAudioItems(libraryId!, options);
      return source === "shows" ? fetchShows(libraryId!, options) : fetchVideoItems(libraryId!, options);
    },
    initialPageParam: 0,
    // Offset paging ends when the next window would start past the reported total. Using the total instead of
    // "page shorter than the page size" keeps the last window reachable when a shelf is an exact multiple of 120
    // (1388 tracks = 11 full windows + 68, but 1200 episodes would otherwise look finished).
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((count, page) => count + page.items.length, 0);
      if (lastPage.total !== null) return loaded < lastPage.total ? loaded : undefined;
      return lastPage.items.length === MEDIA_PAGE_SIZE ? loaded : undefined;
    },
    enabled: libraryId !== undefined,
    staleTime: 60_000,
  });

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];

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
  };
}
