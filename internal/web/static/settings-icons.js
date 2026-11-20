// Which small icon a library kind or module wears in Settings. The drawings live in settings-cards.css.
const kindIcons = { books: "book", comics: "book", manga: "book", movies: "video", tv: "video", anime: "video", audiobooks: "audio", podcasts: "podcast", music: "music", documents: "doc", vault: "doc" };
const moduleIcons = { video: "video", audio: "audio", documents: "doc", narration: "podcast", hardcover: "book", hardcover_progress: "book", hardcover_ratings: "book", editor: "table", android: "phone" };

export const iconForKind = (kind) => kindIcons[kind] || "doc";
export const iconForModule = (id) => moduleIcons[id] || "doc";
