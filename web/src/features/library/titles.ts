// features/library/titles.ts — display normalisation ported verbatim from the legacy book-cards.js (displayTitle
// and seriesIndex), because the server sends the raw catalogue values and the old app fixed them at render time:
//
// 1. A series index is sometimes baked into the title as a zero-padded prefix ("01 Dies the Fire"), which then
//    reads as part of the name and sorts the shelf by a number. Only a zero-padded integer counts as an index —
//    "1984" and "127 Hours" are titles, "01 Dies the Fire" is not.
// 2. Every book row carries series_index = 1 by default, so the stored number only means something when the book
//    actually belongs to a series. Without a series name it is noise, and 1984 is not "Vol. 1".
//
// Doing this here rather than in the new card keeps the rule in one place and identical to the legacy shelf, so the
// two UIs can't disagree about a book's name while both are served by the same server.

export interface TitleFacts {
  title: string;
  /** As `/api/books` names them — snake_case, straight off the wire. */
  series?: string | undefined;
  series_index?: number | undefined;
}

const ZERO_PADDED_PREFIX = /^0(\d{1,2})\s*[-–—:.]?\s+(?=\S)/;

/** The title as a reader would say it, without a leading zero-padded series number. */
export function displayTitle(book: TitleFacts): string {
  const raw = book.title.trim() || "Untitled";
  const match = raw.match(ZERO_PADDED_PREFIX);
  if (!match) return raw;
  const rest = raw.slice(match[0].length).trim();
  return rest.length >= 2 ? rest : raw;
}

/** The volume number to show, or 0 for "don't show one". */
export function displaySeriesIndex(book: TitleFacts): number {
  const stored = Number(book.series_index ?? 0);
  const seriesName = (book.series ?? "").trim();
  const sameAsTitle = seriesName.toLowerCase() === book.title.trim().toLowerCase();
  const inSeries = seriesName !== "" && !sameAsTitle;
  if (inSeries && Number.isFinite(stored) && stored > 0) return stored;
  const prefixed = book.title.match(ZERO_PADDED_PREFIX);
  return prefixed ? Number(prefixed[1]) : 0;
}
