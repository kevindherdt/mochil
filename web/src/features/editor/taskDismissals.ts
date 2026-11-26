// features/editor/taskDismissals.ts — which books each tool has been told to stop offering.
//
// Dismissal used to be one per-book list per library, and that is exactly what made the tools look broken: putting a
// found ISBN aside also hid that book's Hardcover match, so the next run read as "the match search ignored it". The
// answer is now per tool, and this module is where those sets live between the two places that write them — the Tasks
// panel (Discard rest / Restore) and the review panel (discarding one book).
//
// The server's `found` / `dismissed` counts stay the truth for how much is left; the ids here are the optimistic cache
// that makes the books actually disappear from the next staging. They survive a reload because the same 41 found ISBNs
// being offered again after a rebuild reads as the editor refusing an answer it was already given.
import { useSyncExternalStore } from "react";
import { dismissTask, restoreTask } from "./api/editor";

const KEY = "mochil-editor-task-dismissed";
/** The old per-library book list. It hid a book from *every* tool at once — the bug this replaces — so it is dropped
 *  rather than migrated: carrying it over would carry the bug over with it. */
const LEGACY_KEY = "mochil-editor-dismissed";
/** One entry meaning "the whole run", for a Discard rest that did not enumerate its books. */
const EVERYTHING = "*";

/** library id → tool id → book ids ("*" = all of them). */
type ByLibrary = Record<string, Record<string, string[]>>;

const EMPTY: Record<string, string[]> = {};

function read(): ByLibrary {
  try {
    localStorage.removeItem(LEGACY_KEY);
    const raw = localStorage.getItem(KEY);
    if (raw === null) return {};
    const parsed: unknown = JSON.parse(raw);
    return typeof parsed === "object" && parsed !== null ? (parsed as ByLibrary) : {};
  } catch {
    // Private mode, or a value from a build that wrote something else: start empty rather than refuse to edit.
    return {};
  }
}

let cache: ByLibrary = read();
const listeners = new Set<() => void>();

function publish(next: ByLibrary): void {
  cache = next;
  try {
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // Private mode: the answers hold for this session and are asked for again next time.
  }
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function withBooks(current: ByLibrary, libraryId: string, tool: string, books: string[]): ByLibrary {
  const forLibrary = current[libraryId] ?? {};
  const before = forLibrary[tool] ?? [];
  // Already "everything": there is nothing left to add, and adding one book must not read as a narrower answer.
  if (before.includes(EVERYTHING)) return current;
  const added = books.length === 0 ? [EVERYTHING] : books;
  return {
    ...current,
    [libraryId]: { ...forLibrary, [tool]: [...new Set([...before, ...added])] },
  };
}

/** The dismissal sets for one library, as a hook so a component re-renders when another one dismisses something. */
export function useTaskDismissals(libraryId: string | undefined): Record<string, string[]> {
  const all = useSyncExternalStore(subscribe, () => cache);
  return libraryId === undefined ? EMPTY : (all[libraryId] ?? EMPTY);
}

/** The books one tool has been told to stop offering. An entry of "*" means the whole run; the caller checks with
 *  isFullyDismissed, because "everything" is not a list that can be read. */
export function dismissedFor(libraryId: string | undefined, tool: string): string[] {
  if (libraryId === undefined) return [];
  return cache[libraryId]?.[tool] ?? [];
}

export function isFullyDismissed(libraryId: string | undefined, tool: string): boolean {
  return dismissedFor(libraryId, tool).includes(EVERYTHING);
}

/** Put books aside for one tool and tell the server. No books means the whole run, which the server reads the same
 *  way. The local set moves first so the row answers immediately; a refused request puts it back. */
export async function dismissBooks(libraryId: string, tool: string, books: string[]): Promise<void> {
  const before = cache;
  publish(withBooks(cache, libraryId, tool, books));
  try {
    await dismissTask(libraryId, tool, books);
  } catch (error) {
    publish(before);
    throw error;
  }
}

/** Bring back everything one tool had put aside — that tool only. */
export async function restoreDismissals(libraryId: string, tool: string): Promise<void> {
  const before = cache;
  const forLibrary = { ...(cache[libraryId] ?? {}) };
  delete forLibrary[tool];
  publish({ ...cache, [libraryId]: forLibrary });
  try {
    await restoreTask(libraryId, tool);
  } catch (error) {
    publish(before);
    throw error;
  }
}
