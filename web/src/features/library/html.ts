// features/library/html.ts — detail-screen descriptions come from the catalogue (calibre/scraper metadata) as
// HTML, and that HTML is untrusted — a dependency isn't worth adding to render it, so this extracts plain text
// instead of rendering markup at all. `DOMParser.parseFromString` on a detached, never-inserted document doesn't
// execute scripts or fetch resources, so reading `.textContent` back out is a safe way to strip tags.
export function htmlToText(html: string): string {
  const parsed = new DOMParser().parseFromString(html, "text/html");
  return (parsed.body.textContent ?? "").replace(/\s+/g, " ").trim();
}
