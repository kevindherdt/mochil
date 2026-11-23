// features/library/kinds.ts — maps a real library's `kind` string (internal/web's libraryView.Kind, the same
// value library-pills.js's KIND_ICON keyed on) to the glyph a card shows on its artwork. Kept here, not in ui/,
// because which kinds exist is server/domain knowledge, not a presentational concern.
//
// There is deliberately no label/tag here: cards no longer carry a text pill under the author (Kevin, 22 Nov
// 2026 — the kind is an icon on the cover instead), so a kind maps to exactly one thing, its glyph.
import type { ComponentType } from "react";
import { BookOpen, Headphones, Clapperboard, BookImage, FileText, Music2, Mic } from "lucide-react";

export interface KindMeta {
  icon: ComponentType<{ size?: number }>;
}

const KIND_META: Record<string, KindMeta> = {
  books: { icon: BookOpen },
  audio: { icon: Headphones },
  audiobooks: { icon: Headphones },
  video: { icon: Clapperboard },
  movies: { icon: Clapperboard },
  tv: { icon: Clapperboard },
  anime: { icon: Clapperboard },
  manga: { icon: BookImage },
  comics: { icon: BookImage },
  documents: { icon: FileText },
  music: { icon: Music2 },
  podcasts: { icon: Mic },
};

const DEFAULT_META: KindMeta = { icon: BookOpen };

export function kindMeta(kind: string): KindMeta {
  return KIND_META[kind.toLowerCase()] ?? DEFAULT_META;
}

/**
 * The switcher's glyph for a *library*, which needs more than `kind`: this server reports `kind: "books"` for
 * Light Novels, Storyteller and Testbooks alike, so a kind-only lookup drew the same open-book icon on every pill —
 * the mock-up's row is readable partly because each shelf has its own glyph. The legacy library-pills.js solved the
 * same problem by falling back to the library id, and this ports that rule (id patterns first, since they are more
 * specific, then kind).
 */
export function libraryIcon(kind: string, id: string): ComponentType<{ size?: number }> {
  const k = kind.toLowerCase();
  const i = id.toLowerCase();
  if (/audiobook|storyteller|podcast|audio/.test(i)) return k === "podcasts" ? Mic : Headphones;
  if (/movie|series|tv|film|anime/.test(i)) return Clapperboard;
  if (/manga|comic/.test(i)) return BookImage;
  if (/music|album|song/.test(i)) return Music2;
  if (/test|text|scripture|doc|paper/.test(i)) return FileText;
  return kindMeta(k).icon;
}

/** The three card families (plan ui-overhaul-plan.md §4.3: "card families, three not nine"):
 *  - **book** — cover, title, author. Also audiobooks, manga and documents: same 2:3 card.
 *  - **video** — movies/TV/anime. Same card as a book: a poster is 2:3 too, so nothing is gained by a second
 *    component, and the mock-up draws Movies/TV exactly like Books.
 *  - **music** — square artwork, no progress chrome (artwork scarcity, per the plan).
 *  Anything else (a notes vault) isn't a shelf this screen browses. */
export type CardFamily = "book" | "video" | "music" | "none";

const BOOK_FAMILY_KINDS = new Set(["books", "audio", "audiobooks", "manga", "comics", "documents"]);
const VIDEO_FAMILY_KINDS = new Set(["video", "movies", "tv", "anime"]);
const MUSIC_FAMILY_KINDS = new Set(["music", "podcasts"]);

export function cardFamily(kind: string): CardFamily {
  const k = kind.toLowerCase();
  if (BOOK_FAMILY_KINDS.has(k)) return "book";
  if (VIDEO_FAMILY_KINDS.has(k)) return "video";
  if (MUSIC_FAMILY_KINDS.has(k)) return "music";
  return "none";
}

export function isBookFamily(kind: string): boolean {
  return cardFamily(kind) === "book";
}

/** Which endpoint a shelf's items come from. This is *not* the same question as which card draws them: audiobooks
 *  are a book card over the audio store (410 files, 0 catalogue rows), while movies are a book card over the video
 *  store. Getting this wrong is what made Music claim to be empty — /api/books knows nothing about audio. */
export type ShelfSource = "catalogue" | "audio" | "videos" | "shows";

export function shelfSource(kind: string): ShelfSource {
  switch (kind.toLowerCase()) {
    case "audiobooks":
    case "audio":
    case "music":
    case "podcasts":
      return "audio";
    case "movies":
      return "videos";
    // A series library is a list of shows, not of 40 loose episodes.
    case "tv":
    case "anime":
    case "video":
      return "shows";
    default:
      return "catalogue";
  }
}

/** Whether this screen can browse the shelf at all (everything except a notes vault). */
export function isShelvable(kind: string): boolean {
  return cardFamily(kind) !== "none";
}

/** A notes wiki/vault is not a shelf you browse for books. The legacy app's library pill row already excluded
    these (library-pills.js's isNotesShelf) — the new switcher must match, or a vault landed as the first pill
    and as the app's landing view. */
const NOTE_KINDS = new Set(["vault", "notes", "wiki"]);

export function isNoteShelf(kind: string): boolean {
  return NOTE_KINDS.has(kind.toLowerCase());
}

/** Why a shelf this screen still can't render is empty — the shelf's own kind, not one blanket sentence. Only the
    notes vault is left: books/audiobooks/manga share the book card, video shelves share it too (a poster is 2:3
    like a cover), and music has the square MusicCard. */
export function unbuiltReason(kind: string): string {
  switch (kind.toLowerCase()) {
    case "vault":
    case "notes":
    case "wiki":
      return "the notes vault gets its own screen";
    default:
      return "this shelf type isn't built yet";
  }
}
