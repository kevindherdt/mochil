// Quick actions: one tap sends a question together with what Mochil knows (new books, what you are reading, one
// book's details). The context is plain text added to the message, kept short, and shown in the chat as a folded
// "Library context" note so you can always see exactly what she was given. Nothing else leaves the server.

const ENTITIES = { "&amp;": "&", "&lt;": "<", "&gt;": ">", "&quot;": '"', "&#39;": "'", "&nbsp;": " ", "&rsquo;": "’", "&lsquo;": "‘" };

/** A description without markup, cut to a length worth reading. */
export function plainText(html, limit = 700) {
  const text = String(html ?? "").replace(/<[^>]*>/g, " ").replace(/&[#a-z0-9]+;/gi, (entity) => ENTITIES[entity] ?? " ").replace(/\s+/g, " ").trim();
  return text.length > limit ? text.slice(0, limit).replace(/\s+\S*$/, "") + "…" : text;
}

/** One book as one line: title, author, series, and how far you are. */
export function bookLine(book) {
  const authors = (book.authors || []).join(", ");
  const series = book.series ? ` (${book.series}${book.series_index ? ` #${book.series_index}` : ""})` : "";
  const progress = book.reading_progress;
  const state = progress?.finished ? ", finished" : progress?.fraction > 0 ? `, ${Math.round(progress.fraction * 100)}% read` : "";
  return `- ${book.title}${authors ? ` by ${authors}` : ""}${series}${state}`;
}

const section = (title, books, max) => (books.length ? `${title}:\n${books.slice(0, max).map(bookLine).join("\n")}` : "");

/** The text added to a message, in a block the chat knows how to fold away again. */
export const wrapContext = (prompt, context) => (context ? `${prompt}\n\n<library-context>\n${context}\n</library-context>` : prompt);

/** Splits a message back into what you asked and the context that came with it. */
export function splitContext(text) {
  const match = /^([\s\S]*?)\s*<library-context>\n([\s\S]*?)\n<\/library-context>\s*$/.exec(String(text ?? ""));
  return match ? { prompt: match[1], context: match[2] } : { prompt: String(text ?? ""), context: "" };
}

export const ACTIONS = [
  { id: "recent", label: "What's new in my library", hint: "Recently added books", prompt: "What's new in my library? Tell me what stands out among the recently added books." },
  { id: "next", label: "What should I read next", hint: "Uses what you are reading and finished", prompt: "What should I read next? Suggest a few books from my library and say why each suits me." },
  { id: "book", label: "Tell me about a book", hint: "Pick a book to summarise", prompt: "Tell me about this book and what it is about, without spoiling more than the description does.", pick: true },
  { id: "polish", label: "Polish my text", hint: "Paste your text after the request", draft: "Polish this text. Keep my meaning and my voice, fix mistakes, and make it read better:\n\n" },
];

/** Gathers the context for an action from the app's own catalogue. `api` fetches JSON. */
export async function gather(action, api, book) {
  if (action.id === "recent") return section("Recently added books", await api("/api/books?sort=recent&limit=15"), 15);
  if (action.id === "next") {
    const [reading, finished, recent] = await Promise.all([api("/api/reading").catch(() => []), api("/api/reading?finished=true&limit=12").catch(() => []), api("/api/books?sort=recent&limit=12").catch(() => [])]);
    return [section("Reading now", reading, 8), section("Recently finished", finished, 12), section("Recently added", recent, 12)].filter(Boolean).join("\n\n");
  }
  if (action.id === "book" && book) {
    const detail = plainText(book.description);
    return `${bookLine(book)}\nLibrary: ${book.library_name || ""}${book.language ? `, language: ${book.language}` : ""}${detail ? `\nDescription: ${detail}` : "\n(No description is saved for this book.)"}`;
  }
  return "";
}

/** The "+" button, its menu above the composer, and the book picker. */
export function createActions({ $, api, send, input, busy = () => false, document = globalThis.document }) {
  const button = $("#hm-quick"), menu = $("#hm-quickmenu");
  const close = () => { menu.hidden = true; button.setAttribute("aria-expanded", "false"); };
  async function run(action, book) {
    close();
    if (action.draft) { input.value = action.draft; input.dispatchEvent(new Event("input")); input.focus(); input.setSelectionRange(input.value.length, input.value.length); return; }
    if (busy()) return;
    let context = "";
    try { context = await gather(action, api, book); } catch (_) { /* she still gets the question */ }
    await send(wrapContext(action.prompt, context));
  }
  function picker(action) {
    menu.innerHTML = '<label class="hm-q-search"><input type="search" id="hm-q-find" placeholder="Search your books" autocomplete="off" aria-label="Search your books"></label><div class="hm-q-results" id="hm-q-results"></div>';
    const find = $("#hm-q-find"), results = $("#hm-q-results");
    let timer = 0, token = 0;
    find.addEventListener("input", () => {
      clearTimeout(timer);
      timer = setTimeout(async () => {
        const mine = ++token, query = find.value.trim();
        if (query.length < 2) { results.replaceChildren(); return; }
        const books = await api(`/api/books?q=${encodeURIComponent(query)}&limit=8`).catch(() => []);
        if (mine !== token) return;
        results.replaceChildren(...books.map((book) => {
          const row = document.createElement("button");
          row.type = "button"; row.className = "hm-q-item";
          row.textContent = bookLine(book).replace(/^- /, "");
          row.onclick = () => run(action, book);
          return row;
        }));
        if (!books.length) results.textContent = "No books found.";
      }, 220);
    });
    find.focus();
  }
  function open() {
    menu.replaceChildren(...ACTIONS.map((action) => {
      const item = document.createElement("button");
      item.type = "button"; item.className = "hm-q-item";
      item.innerHTML = "<strong></strong><small></small>";
      item.firstChild.textContent = action.label; item.lastChild.textContent = action.hint;
      item.onclick = () => (action.pick ? picker(action) : run(action));
      return item;
    }));
    menu.hidden = false;
    button.setAttribute("aria-expanded", "true");
  }
  button.onclick = () => (menu.hidden ? open() : close());
  document.addEventListener("keydown", (event) => { if (event.key === "Escape" && !menu.hidden) close(); });
  // The path is taken when the click happens, so a menu item that rebuilds the menu does not count as "outside".
  document.addEventListener("click", (event) => { const path = event.composedPath(); if (!menu.hidden && !path.includes(menu) && !path.includes(button)) close(); });
  return { run, open, close, actions: ACTIONS };
}
