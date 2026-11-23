// features/library/useGroups.ts — the series/authors browse lists, paged against the real endpoints. Real
// libraries are big (1083 series in Light Novels), so this pages rather than fetching everything: a "show more"
// button appends the next window, and `hasMore` comes from the last page being full.
import { useInfiniteQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchGroups, GROUP_PAGE_SIZE, type GroupInfo, type GroupKind } from "./api/groups";

export interface GroupsState {
  query: AsyncQueryLike<GroupInfo[]>;
  hasMore: boolean;
  loadMore: () => void;
  isLoadingMore: boolean;
}

export function useGroups(kind: GroupKind, libraryId: string | undefined): GroupsState {
  const query = useInfiniteQuery({
    queryKey: ["library", "groups", kind, libraryId],
    queryFn: ({ pageParam }) => fetchGroups(kind, libraryId!, pageParam, GROUP_PAGE_SIZE),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => (lastPage.hasMore ? allPages.length * GROUP_PAGE_SIZE : undefined),
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
