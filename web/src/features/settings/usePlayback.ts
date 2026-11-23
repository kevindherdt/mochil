// features/settings/usePlayback.ts / useDiagnostics — kept in one file because both are read-and-show panels whose
// hooks are a handful of lines; splitting them would create two files that only ever change together.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchPlayback, fetchPlaybackPolicy, savePlayback, type PlaybackSettings } from "./api/playback";
import { fetchDiagnostics, type Diagnostics } from "./api/diagnostics";

export function usePlayback(): AsyncQueryLike<PlaybackSettings> {
  const query = useQuery({ queryKey: ["settings", "playback"], queryFn: fetchPlayback, staleTime: 10_000 });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

/** What players are actually told. Separate from `usePlayback` because it is a different endpoint with its own
 *  fallbacks — if the two ever disagree, that is worth seeing. */
export function usePlaybackPolicy(): AsyncQueryLike<{ wifiQuality: string; mobileQuality: string }> {
  const query = useQuery({ queryKey: ["settings", "playback-policy"], queryFn: fetchPlaybackPolicy, staleTime: 10_000 });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useSavePlayback(): {
  save: (settings: PlaybackSettings, onDone?: () => void) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
} {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: savePlayback,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["settings", "playback"] });
      void queryClient.invalidateQueries({ queryKey: ["settings", "playback-policy"] });
    },
  });
  return {
    save: (settings, onDone) => mutation.mutate(settings, { onSuccess: () => onDone?.() }),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: () => mutation.reset(),
  };
}

export function useDiagnostics(enabled: boolean): AsyncQueryLike<Diagnostics> {
  const query = useQuery({
    queryKey: ["settings", "diagnostics"],
    queryFn: fetchDiagnostics,
    enabled,
    // A snapshot of a moving thing: never show a cached one as if it were now.
    staleTime: 0,
    gcTime: 0,
    refetchOnMount: "always",
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}
