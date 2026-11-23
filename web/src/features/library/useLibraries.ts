// features/library/useLibraries.ts — the hook for the real library switcher (PillRow). Real library list
// changes rarely (Settings -> Libraries), so a longer staleTime is fine — unlike /api/version.
import { useQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchLibraries, type LibraryInfo } from "./api/libraries";

export function useLibraries(): AsyncQueryLike<LibraryInfo[]> {
  const query = useQuery({
    queryKey: ["library", "libraries"],
    queryFn: fetchLibraries,
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
