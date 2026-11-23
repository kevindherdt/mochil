// features/notes/useNotes.ts — Notes' data and mutations (plan docs/rewrite-progress.md §2.7), same one-query/
// several-mutations shape as settings/useAccounts.ts.
//
// Two queries, deliberately separate: the vault *list* (which library to read at all) and that library's *entries*
// (folders + pages). They change for different reasons — a second vault can be added without a single page
// changing — and the entries query stays disabled until a library id actually exists, or the first render would
// ask `/api/documents?library=undefined`.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import {
  createFolder,
  createPage,
  listEntries,
  listVaultLibraries,
  readPage,
  savePage,
  trashPage,
  type VaultEntry,
} from "./api/vault";

const VAULTS_KEY = ["notes", "vaults"] as const;
const entriesKey = (library: string) => ["notes", "entries", library] as const;

/** The vault libraries, and which one this screen is showing. `undefined` while the first request is in flight. */
export function useVaultLibrary(): string | undefined {
  const query = useQuery({ queryKey: VAULTS_KEY, queryFn: listVaultLibraries, staleTime: 60_000 });
  return query.data?.[0]?.id;
}

export function useVaultEntries(library: string | undefined): AsyncQueryLike<VaultEntry[]> {
  const query = useQuery({
    queryKey: entriesKey(library ?? ""),
    queryFn: () => listEntries(library as string),
    enabled: library !== undefined,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

interface Mutations {
  create: (input: { library: string; path: string; content: string }, onDone?: () => void) => void;
  save: (
    input: { library: string; itemId: string; content: string; revision: string },
    onDone?: (revision: string) => void,
  ) => void;
  remove: (input: { library: string; itemId: string; revision: string }, onDone?: () => void) => void;
  addFolder: (input: { library: string; path: string }, onDone?: () => void) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
}

export function useNotesMutations(): Mutations {
  const queryClient = useQueryClient();
  const invalidate = (library: string) => {
    void queryClient.invalidateQueries({ queryKey: entriesKey(library) });
    // The library list carries a `count`, which a create or a trash changes — so it is invalidated too, or the
    // Settings → Libraries row would keep saying 48 items after the 49th note.
    void queryClient.invalidateQueries({ queryKey: VAULTS_KEY });
    void queryClient.invalidateQueries({ queryKey: ["library", "libraries"] });
  };

  const create = useMutation({
    mutationFn: (input: { library: string; path: string; content: string }) =>
      createPage(input.library, input.path, input.content),
    onSuccess: (_result, input) => invalidate(input.library),
  });
  const save = useMutation({
    mutationFn: (input: { library: string; itemId: string; content: string; revision: string }) =>
      savePage(input.library, input.itemId, input.content, input.revision),
    onSuccess: (_result, input) => invalidate(input.library),
  });
  const remove = useMutation({
    mutationFn: (input: { library: string; itemId: string; revision: string }) =>
      trashPage(input.library, input.itemId, input.revision),
    onSuccess: (_result, input) => invalidate(input.library),
  });
  const folder = useMutation({
    mutationFn: (input: { library: string; path: string }) => createFolder(input.library, input.path),
    onSuccess: (_result, input) => invalidate(input.library),
  });

  const errors = [create.error, save.error, remove.error, folder.error].filter(
    (error) => error !== null && error !== undefined,
  );

  return {
    create: (input, onDone) => create.mutate(input, { onSuccess: () => onDone?.() }),
    save: (input, onDone) => save.mutate(input, { onSuccess: (revision) => onDone?.(revision) }),
    remove: (input, onDone) => remove.mutate(input, { onSuccess: () => onDone?.() }),
    addFolder: (input, onDone) => folder.mutate(input, { onSuccess: () => onDone?.() }),
    isPending: create.isPending || save.isPending || remove.isPending || folder.isPending,
    error: errors[0] ?? null,
    reset: () => {
      create.reset();
      save.reset();
      remove.reset();
      folder.reset();
    },
  };
}

/** Loads one page's markdown body on demand — the entry list only carries metadata. */
export async function loadPage(textUrl: string) {
  return readPage(textUrl);
}
