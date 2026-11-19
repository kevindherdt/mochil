// Moving or copying books to another library: what the dialogs say, kept apart from the page so it can be tested.

const plural = (count, word) => `${count} ${word}${count === 1 ? "" : "s"}`;

export const transferVerb = (mode) => (mode === "move" ? { do: "Move", done: "Moved", noun: "move" } : { do: "Copy", done: "Copied", noun: "copy" });

// transferRequest is the body for the server; preview asks what would happen without doing it.
export const transferRequest = (library, to, ids, mode, preview) => ({ library, to, ids, mode, preview });

// blockedText is one short sentence per problem, the first few only, for the dialog's error line.
export function blockedText(preview, limit = 3) {
  const blocked = (preview.books || []).filter((book) => book.reason);
  if (!blocked.length) return "";
  const lines = blocked.slice(0, limit).map((book) => `“${book.title}”: ${book.reason}`);
  const more = blocked.length > limit ? ` …and ${plural(blocked.length - limit, "more")}.` : "";
  return `${plural(blocked.length, "book")} cannot go: ${lines.join(" ")}${more} Nothing was changed.`;
}

// confirmText explains what is about to happen, including what travels with a move.
export function confirmText(mode, count, toName) {
  const verb = transferVerb(mode);
  const books = plural(count, "book");
  return mode === "move"
    ? `${verb.do} ${books} to “${toName}”? Their corrections, custom columns and everyone’s reading place go with them, and they leave this library. You can undo it right after.`
    : `${verb.do} ${books} to “${toName}”? The originals stay here; the copies start without a reading place. You can undo it right after.`;
}

export const doneText = (mode, count, toName) => `${transferVerb(mode).done} ${plural(count, "book")} to “${toName}”.`;
