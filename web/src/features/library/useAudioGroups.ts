// features/library/useAudioGroups.ts — the audio equivalent of useGroups.ts, paged the same way (a full page
// probably has a next one; Show more appends it).
import { useInfiniteQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { AUDIO_GROUP_PAGE_SIZE, fetchAudioGroups, type AudioGroupInfo, type AudioGroupKind } from "./api/audioGroups";

export interface AudioGroupsState {
  query: AsyncQueryLike<AudioGroupInfo[]>;
  hasMore: boolean;
  loadMore: () => void;
  isLoadingMore: boolean;
}

export function useAudioGroups(libraryId: string | undefined, groupType: AudioGroupKind | undefined, q = ""): AudioGroupsState {
  const query = useInfiniteQuery({
    queryKey: ["library", "audio-groups", libraryId, groupType, q],
    queryFn: ({ pageParam }) => fetchAudioGroups(libraryId!, groupType!, q, pageParam, AUDIO_GROUP_PAGE_SIZE),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => (lastPage.hasMore ? allPages.length * AUDIO_GROUP_PAGE_SIZE : undefined),
    enabled: libraryId !== undefined && groupType !== undefined,
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
