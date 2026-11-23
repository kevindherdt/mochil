// features/settings/useNarration.ts — the Narrate panel's data: the saved settings, the runtime's real status, and
// the save. Status is a separate query on purpose — it changes while the panel is open (the voice loads on first
// use), and it is the half that answers "why is this not working".
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchNarration, fetchNarrationStatus, saveNarration, type NarrationSettings, type NarrationStatus, type NarrationUpdate } from "./api/narration";

export function useNarration(enabled: boolean): AsyncQueryLike<NarrationSettings> {
  const query = useQuery({ queryKey: ["settings", "narration"], queryFn: fetchNarration, enabled, staleTime: 10_000 });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useNarrationStatus(enabled: boolean): AsyncQueryLike<NarrationStatus> {
  const query = useQuery({
    queryKey: ["settings", "narration-status"],
    queryFn: fetchNarrationStatus,
    enabled,
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

export function useSaveNarration(): {
  save: (update: NarrationUpdate, onDone?: () => void) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
} {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: saveNarration,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["settings", "narration"] });
      void queryClient.invalidateQueries({ queryKey: ["settings", "narration-status"] });
    },
  });
  return {
    save: (update, onDone) => mutation.mutate(update, { onSuccess: () => onDone?.() }),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: () => mutation.reset(),
  };
}
