// features/library/useMediaDetail.ts — single-item fetches for the four detail screens, each a plain useQuery
// (no paging here — these resolve to one item, not a shelf). Disabled until every path param it needs exists,
// so TanStack Router's loose typing on route params can't send `undefined` through to a `${undefined}` URL.
import { useQuery } from "@tanstack/react-query";
import { fetchBookDetail } from "./api/books";
import { fetchAudioDetail, fetchEpisodes, fetchShowDetail, fetchVideoDetail } from "./api/mediaDetail";
import { fetchAudioGroupTracks, type AudioGroupKind } from "./api/audioGroups";

export function useBookDetail(libraryId: string | undefined, bookId: string | undefined) {
  return useQuery({
    queryKey: ["library", "book-detail", libraryId, bookId],
    queryFn: () => fetchBookDetail(libraryId!, bookId!),
    enabled: libraryId !== undefined && bookId !== undefined,
  });
}

export function useVideoDetail(libraryId: string | undefined, itemId: string | undefined) {
  return useQuery({
    queryKey: ["library", "video-detail", libraryId, itemId],
    queryFn: () => fetchVideoDetail(libraryId!, itemId!),
    enabled: libraryId !== undefined && itemId !== undefined,
  });
}

export function useShowDetail(libraryId: string | undefined, itemId: string | undefined) {
  return useQuery({
    queryKey: ["library", "show-detail", libraryId, itemId],
    queryFn: () => fetchShowDetail(libraryId!, itemId!),
    enabled: libraryId !== undefined && itemId !== undefined,
  });
}

export function useEpisodes(libraryId: string | undefined, showSourceName: string | undefined, season: number | undefined) {
  return useQuery({
    queryKey: ["library", "episodes", libraryId, showSourceName, season],
    queryFn: () => fetchEpisodes(libraryId!, showSourceName!, season),
    enabled: libraryId !== undefined && showSourceName !== undefined,
  });
}

export function useAudioDetail(libraryId: string | undefined, itemId: string | undefined) {
  return useQuery({
    queryKey: ["library", "audio-detail", libraryId, itemId],
    queryFn: () => fetchAudioDetail(libraryId!, itemId!),
    enabled: libraryId !== undefined && itemId !== undefined,
  });
}

export function useAudioGroupTracks(libraryId: string | undefined, groupType: AudioGroupKind | undefined, groupId: string | undefined) {
  return useQuery({
    queryKey: ["library", "audio-group-tracks", libraryId, groupType, groupId],
    queryFn: () => fetchAudioGroupTracks(libraryId!, groupType!, groupId!),
    enabled: libraryId !== undefined && groupType !== undefined && groupId !== undefined,
  });
}
