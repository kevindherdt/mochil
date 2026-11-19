// The status column: one pill per finding, coloured by how much it matters. The glyph repeats the colour
// (e-ink and colour-blind readers cannot rely on it): ✕ missing, ⚠ check, ✓ fine, ✎ edited.
const TONE = {
  no_author: "bad", no_isbn: "bad", no_cover: "bad", unreadable: "bad",
  duplicate: "warn", no_number: "warn", no_description: "warn", hardcover_doubt: "warn", match_unsure: "warn", isbn_unsure: "warn",
  isbn_found: "good", match_found: "good",
};
// Findings a Hardcover lookup can often fill in: they get a Fetch button.
const FETCHABLE = new Set(["no_isbn", "no_description", "no_author", "no_cover"]);

export const toneOf = (issue) => TONE[issue] || "quiet";
export const canFetch = (issue) => FETCHABLE.has(issue);

export function badgeHTML(issue, text, rowId, escapeHTML) {
  const label = escapeHTML(text);
  const fetch = canFetch(issue) ? `<button type="button" class="ed-fetch" data-fetch="${escapeHTML(rowId)}" title="Look this book up on Hardcover">Fetch</button>` : "";
  return `<span class="ed-badge" data-tone="${toneOf(issue)}">${label}${fetch}</span>`;
}

export const okBadgeHTML = () => '<span class="ed-badge" data-tone="good">OK</span>';
