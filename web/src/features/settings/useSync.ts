// features/settings/useSync.ts — the Reading-sync and Metadata-provider hooks. Both panels are one section of the
// settings document, so they share this file; splitting them would produce two near-identical hooks.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchSync, saveSync, type SyncSettings, type SyncUpdate } from "./api/sync";
import { fetchProviders, saveProviderTokens, type ProviderSettings } from "./api/providers";

export function useSync(enabled: boolean): AsyncQueryLike<SyncSettings> {
  const query = useQuery({ queryKey: ["settings", "sync"], queryFn: fetchSync, enabled, staleTime: 10_000 });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useSaveSync(): {
  save: (update: SyncUpdate, onDone?: () => void) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
} {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: saveSync,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["settings", "sync"] });
    },
  });
  return {
    save: (update, onDone) => mutation.mutate(update, { onSuccess: () => onDone?.() }),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: () => mutation.reset(),
  };
}

export function useProviders(enabled: boolean): AsyncQueryLike<ProviderSettings> {
  const query = useQuery({ queryKey: ["settings", "providers"], queryFn: fetchProviders, enabled, staleTime: 10_000 });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export function useSaveProviderTokens(): {
  save: (input: { hardcover: string; tmdb: string }, onDone?: () => void) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
} {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: saveProviderTokens,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["settings", "providers"] });
    },
  });
  return {
    save: (input, onDone) => mutation.mutate(input, { onSuccess: () => onDone?.() }),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: () => mutation.reset(),
  };
}
