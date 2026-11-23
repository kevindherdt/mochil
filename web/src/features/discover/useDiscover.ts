// features/discover/useDiscover.ts — the two rails Discover is built from: what the reader is in the middle of, and
// what arrived most recently. Both are existing endpoints; Discover is a view over them, not a new subsystem.
import { useQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchRecentBooks, type BookInfo } from "../library/api/books";
import { fetchContinueReading, type ReadingItem } from "./api/reading";
import { fetchContinueListening, fetchContinueWatching, type ContinueAudio, type ContinueVideo } from "./api/progress";

export function useContinueReading(limit = 20): AsyncQueryLike<ReadingItem[]> {
  const query = useQuery({
    queryKey: ["discover", "continue", limit],
    queryFn: () => fetchContinueReading(limit),
    // Reading progress changes as you read; a stale "continue" rail is exactly the thing the old dashboard got
    // wrong, so this is refetched on focus like any other live list.
    staleTime: 15_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useRecentlyAdded(limit = 20): AsyncQueryLike<BookInfo[]> {
  const query = useQuery({
    queryKey: ["discover", "recent", limit],
    queryFn: () => fetchRecentBooks(limit),
    staleTime: 60_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useContinueListening(limit = 12): AsyncQueryLike<ContinueAudio[]> {
  const query = useQuery({
    queryKey: ["discover", "listening", limit],
    queryFn: () => fetchContinueListening(limit),
    // Same reasoning as Continue reading: half-heard is a live state, so it refetches on focus.
    staleTime: 15_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useContinueWatching(limit = 12): AsyncQueryLike<ContinueVideo[]> {
  const query = useQuery({
    queryKey: ["discover", "watching", limit],
    queryFn: () => fetchContinueWatching(limit),
    staleTime: 15_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}
