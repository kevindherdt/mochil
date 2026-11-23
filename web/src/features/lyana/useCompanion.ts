// features/lyana/useCompanion.ts — the AI companion's identity and preferences.
//
// Kevin named his "Lyana", and pointed out the obvious consequence: "this is how i named my companion model but
// people should be able to use whatever name they want, so for the button and header it would be better named Ai
// instead of Lyana Ai". So the navigation says **AI**, and the name is a setting with the neutral default "AI".
//
// Where the name lives, and why it is not on the server: `/api/settings` is the server's config *document*
// (libraries, sync user, public address). It has no assistant section, and inventing one here would either be
// dropped on the next config save or need a backend schema change. A per-browser preference is the honest scope
// for now — the same choice the legacy app made with its session id. When a real per-account preferences store
// exists, this module is the one place that changes.
import { useQuery } from "@tanstack/react-query";
import { useSyncExternalStore } from "react";
import { fetchAvatarState, fetchNarrationStatus, type AvatarState, type NarrationStatus } from "./api/companion";

const NAME_KEY = "mochil:ai-name";
const HANDS_FREE_KEY = "mochil:ai-hands-free";

/** Listeners so every component that shows the name re-renders when it changes (no context plumbing for one string). */
const listeners = new Set<() => void>();
const announce = () => listeners.forEach((listener) => listener());

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function readStored(key: string): string {
  try {
    return window.localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

function writeStored(key: string, value: string): void {
  try {
    if (value === "") window.localStorage.removeItem(key);
    else window.localStorage.setItem(key, value);
  } catch {
    // Private browsing or a full quota — a preference that cannot be saved must not break the page.
  }
  announce();
}

/** The display name. Empty means "not chosen yet" → the neutral default. */
export function useCompanionName(): string {
  const stored = useSyncExternalStore(
    subscribe,
    () => readStored(NAME_KEY),
    () => "",
  );
  return stored.trim() === "" ? "AI" : stored.trim();
}

export function setCompanionName(name: string): void {
  writeStored(NAME_KEY, name.trim());
}

/** Hands-free: after a spoken message is transcribed it is sent without pressing send. */
export function useHandsFree(): boolean {
  const stored = useSyncExternalStore(
    subscribe,
    () => readStored(HANDS_FREE_KEY),
    () => "",
  );
  return stored === "1";
}

export function setHandsFree(enabled: boolean): void {
  writeStored(HANDS_FREE_KEY, enabled ? "1" : "");
}

/** The server's narration state: which voices exist, whether the mic can work at all. */
export function useNarration(): { data: NarrationStatus | undefined; isLoading: boolean; refetch: () => void } {
  const query = useQuery({ queryKey: ["lyana", "narration"], queryFn: fetchNarrationStatus, staleTime: 60_000 });
  return { data: query.data, isLoading: query.isLoading, refetch: () => void query.refetch() };
}

/** The avatar the companion is drawn as, and the voice it speaks with. */
export function useAvatar(): { data: AvatarState | undefined; refetch: () => void } {
  const query = useQuery({ queryKey: ["lyana", "avatar"], queryFn: fetchAvatarState, staleTime: 60_000 });
  return { data: query.data, refetch: () => void query.refetch() };
}
