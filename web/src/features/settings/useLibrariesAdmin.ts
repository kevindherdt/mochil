// features/settings/useLibrariesAdmin.ts — the Libraries panel's data: the config entries, the folder browser, and
// the three mutations (add, remove, scan).
//
// Deliberately separate from features/library/useLibraries, which is the *browsing* hook every reader screen uses
// (`/api/libraries` — names, kinds, item counts). This one is administration: it reads the settings document, which
// is administrator-only, and it is the only place that writes it.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchSettings } from "./api/settings";
import {
  addLibrary,
  fetchFolders,
  readLibraryEntries,
  removeLibrary,
  scanLibrary,
  updateLibrary,
  type FolderListing,
  type LibraryEntry,
  type LibraryOptions,
  type NewLibrary,
} from "./api/libraries";
import type { SettingsApplyResult } from "./api/settings";

/** What the folder chooser just inspected, so the panel can prefill and explain instead of asking blind questions. */
export interface PendingFolder {
  path: string;
  suggestedName: string;
  suggestedKind: string;
  storage: string;
  evidence: string[];
}

export interface LibraryAdminState {
  entries: LibraryEntry[];
  /** The whole settings document as the panel last read it — kept so a save can report what the server applied. */
  document: Record<string, unknown>;
}

export function useLibraryAdmin(enabled: boolean): AsyncQueryLike<LibraryAdminState> {
  const query = useQuery({
    queryKey: ["settings", "library-admin"],
    queryFn: async (): Promise<LibraryAdminState> => {
      const document = await fetchSettings();
      return { entries: readLibraryEntries(document), document };
    },
    enabled,
    staleTime: 10_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

/** The folder browser's listing for one path (empty string = the configured roots). A real query rather than local
 *  state so re-entering a folder reuses the answer instead of walking the disk again. */
export function useFolders(path: string | undefined): AsyncQueryLike<FolderListing> {
  const query = useQuery({
    queryKey: ["settings", "folders", path ?? null],
    queryFn: () => fetchFolders(path ?? ""),
    enabled: path !== undefined,
    staleTime: 30_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export interface LibraryMutations {
  add: (input: NewLibrary, onDone?: (result: SettingsApplyResult) => void) => void;
  remove: (id: string, onDone?: (result: SettingsApplyResult) => void) => void;
  scan: (id: string) => void;
  /** Rename a library or change its kind. Every other field on its config entry is preserved. */
  update: (
    input: { id: string; name?: string; kind?: string; options?: LibraryOptions },
    onDone?: (result: SettingsApplyResult) => void,
  ) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
}

export function useLibraryMutations(): LibraryMutations {
  const queryClient = useQueryClient();
  // A library change moves three things at once: the administrator view, the browsing list every reader screen
  // reads, and the item counts on them.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["settings", "library-admin"] });
    void queryClient.invalidateQueries({ queryKey: ["library", "libraries"] });
  };

  const add = useMutation({
    mutationFn: addLibrary,
    onSuccess: (result) => {
      invalidate();
      return result;
    },
  });
  const remove = useMutation({ mutationFn: removeLibrary, onSuccess: invalidate });
  const update = useMutation({
    mutationFn: (input: { id: string; name?: string; kind?: string; options?: LibraryOptions }) =>
      updateLibrary(input.id, {
        ...(input.name === undefined ? {} : { name: input.name }),
        ...(input.kind === undefined ? {} : { kind: input.kind }),
        ...(input.options === undefined ? {} : { options: input.options }),
      }),
    onSuccess: invalidate,
  });
  const scan = useMutation({ mutationFn: scanLibrary });

  const errors = [add.error, remove.error, scan.error, update.error].filter((error) => error !== null && error !== undefined);

  return {
    add: (input, onDone) => add.mutate(input, { onSuccess: (result) => onDone?.(result) }),
    remove: (id, onDone) => remove.mutate(id, { onSuccess: (result) => onDone?.(result) }),
    scan: (id) => scan.mutate(id),
    update: (input, onDone) =>
      update.mutate(input, { onSuccess: (result) => onDone?.(result as SettingsApplyResult) }),
    isPending: add.isPending || remove.isPending || scan.isPending || update.isPending,
    error: errors[0] ?? null,
    reset: () => {
      add.reset();
      remove.reset();
      scan.reset();
      update.reset();
    },
  };
}
