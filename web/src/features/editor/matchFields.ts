// Which fields a Hardcover match is allowed to write, remembered per library.
//
// A match carries a title, an author list, a series, a position, a year and a blurb, and "take the match" used to mean
// all of them. Sometimes the point is the blurb and the series name is the one thing that must not move — so the fields
// are a choice. The choice belongs to the library, not to the browser that made it: it lives on the server, so the same
// person sees the same boxes on another machine and a browser that has never opened the editor does not offer to move a
// series name this library pinned. Nothing here writes a book: it narrows what a review can offer, and the review still
// shows every change before it is saved.
import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchMatchFields, saveMatchFields, type EditorField } from "./api/editor";

/** The fields a match can carry. Publisher, language, tags and the sort keys are not among them. */
export const MATCH_WRITABLE: readonly EditorField[] = [
  "title",
  "authors",
  "series",
  "series_index",
  "published",
  "description",
  "identifiers",
];

/** The key the server's answer is cached under, so a save can write the same entry it later reads. */
function queryKeyFor(libraryId: string | undefined): readonly [string, string, string] {
  return ["editor", "match-fields", libraryId ?? ""] as const;
}

/** Nothing saved yet — or still loading — means every field, which is what a match did before this choice existed.
 *  One shared array, because the snapshot is compared by identity. */
const EVERY_FIELD: EditorField[] = [...MATCH_WRITABLE];

/** What each library last answered, or the choice just made optimistically. */
const saved = new Map<string, EditorField[]>();
/** Stable snapshots per library, so a read does not look like a change. */
const resolved = new Map<string, EditorField[]>();
const listeners = new Set<() => void>();

/** The server's answer, or a choice the person just made, in the editor's own order and without anything a match
 *  cannot write — the tools only ever offer the seven, and a name from an older server must not leak into staging. */
function canonical(fields: readonly string[]): EditorField[] {
  return MATCH_WRITABLE.filter((field) => fields.includes(field));
}

/** Replace one library's snapshot and tell the editor. Returns the array now held, so a failed write can tell whether
 *  its own choice is still the current one before rolling it back. */
function publish(libraryId: string, fields: readonly string[]): EditorField[] {
  const previous = matchFieldsFor(libraryId);
  const next = canonical(fields);
  if (previous.length === next.length && previous.every((field, index) => field === next[index])) {
    // Same answer, so keep the array: useSyncExternalStore compares by identity and a new copy would re-render.
    return previous;
  }
  saved.set(libraryId, next);
  resolved.set(libraryId, next);
  for (const listener of listeners) listener();
  return next;
}

/** The fields this library's matches may write. Absent, unanswered or unreadable means all of them: reading a failed
 *  request as "nothing" would stage no changes at all and look like the match found nothing. */
export function matchFieldsFor(libraryId: string | undefined): EditorField[] {
  const key = libraryId ?? "*";
  const known = resolved.get(key);
  if (known !== undefined) return known;
  const stored = libraryId === undefined ? undefined : saved.get(libraryId);
  const value = stored ?? EVERY_FIELD;
  resolved.set(key, value);
  return value;
}

/** Choose the fields for one library. The snapshot moves immediately and the server is told in the background; a
 *  refused write puts the previous choice back, so a box the server did not accept does not stay ticked. */
export function setMatchFieldsFor(libraryId: string, fields: EditorField[]): void {
  const previous = matchFieldsFor(libraryId);
  const attempt = publish(libraryId, fields);
  void saveMatchFields(libraryId, canonical(fields)).catch(() => {
    // Only roll back if nothing newer was chosen while this write was in flight.
    if (matchFieldsFor(libraryId) === attempt) publish(libraryId, previous);
  });
}

export function useMatchFields(libraryId: string | undefined): [EditorField[], (fields: EditorField[]) => void] {
  const client = useQueryClient();
  const queryKey = useMemo(() => queryKeyFor(libraryId), [libraryId]);
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => fetchMatchFields(libraryId ?? "", signal),
    enabled: libraryId !== undefined,
  });

  // The server is the answer: publish it once it arrives, so the boxes agree with the file the next editor start reads.
  const answer = query.data;
  useEffect(() => {
    if (libraryId !== undefined && answer !== undefined) publish(libraryId, answer);
  }, [libraryId, answer]);

  const { mutate } = useMutation({
    mutationFn: (fields: EditorField[]) => saveMatchFields(libraryId ?? "", fields),
    // Optimistic: the checkbox follows the click, and the request decides afterwards whether that was true.
    onMutate: (fields) => {
      const previous = matchFieldsFor(libraryId);
      const attempt = publish(libraryId ?? "", fields);
      client.setQueryData(queryKey, fields);
      return { previous, attempt };
    },
    onError: (_error, _fields, context) => {
      if (libraryId === undefined || context === undefined) return;
      if (matchFieldsFor(libraryId) !== context.attempt) return;
      publish(libraryId, context.previous);
      client.setQueryData(queryKey, context.previous);
    },
  });

  const subscribe = useCallback((listener: () => void) => {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }, []);
  const snapshot = useSyncExternalStore(
    subscribe,
    () => matchFieldsFor(libraryId),
    () => matchFieldsFor(libraryId),
  );
  const update = useCallback(
    (fields: EditorField[]) => {
      if (libraryId !== undefined) mutate(fields);
    },
    [libraryId, mutate],
  );
  // The snapshot is never mutated in place; it is typed mutable only because that is the surface this module has
  // always returned, and EditorTools narrows it with filter/map rather than editing it.
  return [snapshot, update];
}
