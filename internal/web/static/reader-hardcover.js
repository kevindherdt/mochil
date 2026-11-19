// A line in the reader's Contents panel saying what this book is matched to on Hardcover, so a wrong match can be
// spotted while reading. It stays hidden unless the Hardcover module is on and the reader may see it.
export function hardcoverLine(reply) {
  if (!reply || !reply.enabled) return null;
  const match = reply.match;
  if (match) {
    const how = match.how === "linked" ? "linked by you" : "matched by name";
    const check = match.doubt ? " · check this match: several Hardcover books fit" : "";
    return { text: `Hardcover: ${match.hardcover_title || `book ${match.hardcover_id}`} (${how})${check}`, href: match.url || "" };
  }
  if (reply.skipped) return { text: "Hardcover: not matched. Link it in Settings → Metadata → Hardcover shelf.", href: "" };
  return { text: "Hardcover: nothing sent yet. It is sent once you read a page.", href: "" };
}

// What a book would be matched to if it were sent now (nothing is sent by looking).
export function previewLine(reply) {
  if (!reply) return { text: "Hardcover could not be reached. Try again later.", href: "" };
  const match = reply.match;
  if (match) {
    const by = (match.authors || []).join(", ");
    const note = reply.linked ? " (linked by you)" : reply.ambiguous ? " (several Hardcover books fit: check this)" : "";
    return { text: `Would match: ${match.title}${by ? ` by ${by}` : ""}${note}`, href: match.source_url || "" };
  }
  const closest = (reply.candidates || []).slice(0, 3).map((candidate) => candidate.title);
  return { text: `No clear match on Hardcover.${closest.length ? ` Closest: ${closest.join("; ")}.` : ""} It can be linked in Settings → Metadata → Hardcover shelf.`, href: "" };
}

export function createReaderHardcover({ document, library, id, fetchImpl = globalThis.fetch }) {
  const line = document.querySelector("#hardcover-line");
  async function refresh() {
    if (!line) return;
    let reply = null;
    try {
      const response = await fetchImpl(`/api/metadata/hardcover/match?library=${encodeURIComponent(library)}&book=${encodeURIComponent(id)}`, { cache: "no-store" });
      if (response.ok) reply = await response.json();
    } catch (_) {}
    const shown = hardcoverLine(reply);
    line.hidden = !shown;
    if (!shown) return;
    show(shown);
    // Nothing sent yet: offer to look the book up, so the match can be checked before reading.
    if (!reply.match) {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = "Check match";
      button.onclick = check;
      line.append(" ", button);
    }
  }

  function show(shown) {
    line.textContent = shown.text;
    if (shown.href) {
      const link = document.createElement("a");
      link.href = shown.href;
      link.target = "_blank";
      link.rel = "noopener noreferrer";
      link.textContent = "View";
      line.append(" · ", link);
    }
  }

  async function check() {
    line.textContent = "Looking on Hardcover…";
    let reply = null;
    try {
      const response = await fetchImpl(`/api/metadata/hardcover/preview?library=${encodeURIComponent(library)}&book=${encodeURIComponent(id)}`, { cache: "no-store" });
      if (response.ok) reply = await response.json();
    } catch (_) {}
    show(previewLine(reply));
  }
  return { refresh, check };
}
