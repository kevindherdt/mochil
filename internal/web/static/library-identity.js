// A library's colour is its identity, not its kind.
//
// Colour used to come from the kind alone, so every books library -- books, manga, light novels,
// notes -- wore the same amber, and a switcher with eight libraries read as one warm smear. The
// icon still says what kind of library it is; the colour now says which library it is, and it is
// derived from the id so it never moves between visits and needs no storage.
//
// The hue list is curated rather than a hash-to-360 spread, because a uniform spread puts muddy
// yellow-greens next to each other. These twelve are evenly spaced around the wheel and all hold up
// as a thin 20px stroke on both a dark and a light surface, where the lightness is set in CSS.

const HUES = [
  10,  // coral
  30,  // amber
  48,  // gold
  88,  // lime
  152, // green
  172, // teal
  194, // sky
  214, // blue
  246, // indigo
  274, // violet
  308, // orchid
  336, // pink
];

export function libraryHue(id) {
  const text = String(id ?? "");
  // FNV-1a for the initial mix, then the fmix32 avalanche on top. FNV alone is not enough here: its
  // low bits barely move for ids of the same length, and eight real library names collapsed onto
  // four hues in a test. The avalanche is what turns one changed character into an unrelated hue.
  let hash = 0x811c9dc5;
  for (let i = 0; i < text.length; i += 1) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  hash ^= hash >>> 16;
  hash = Math.imul(hash, 0x85ebca6b) >>> 0;
  hash ^= hash >>> 13;
  hash = Math.imul(hash, 0xc2b2ae35) >>> 0;
  hash ^= hash >>> 16;
  // xor is signed 32-bit in JS, so a negative hash would index past the end of the list.
  return HUES[(hash >>> 0) % HUES.length];
}

/** The destinations that are not libraries keep their neutral treatment. */
const NOT_LIBRARIES = new Set(["", "home", "settings", "editor", "hermes"]);

export function isLibraryDestination(value) {
  const text = String(value ?? "");
  // The switcher's own entries are sentinels ("__home__", "__settings__") rather than library ids,
  // and giving them a library's colour made Home look like a shelf.
  if (text.startsWith("__")) return false;
  return !NOT_LIBRARIES.has(text);
}
