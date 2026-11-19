// Switching Covers ⇄ List replaces the whole shelf, so the person's place has
// to be carried across: remember the card at the top of the viewport, then put
// that same card back at the same height once the new layout has rendered.
// (Same idea as catalogue-return.js, which does it for opened series.)

const CONTAINERS = ["#books", "#series-index", "#collection-index"];
const CARDS = ".book[data-book], .series-card, .series-row, .author-card, .author-row";

function placeKey(card) {
  const dataset = card.dataset || {};
  if (dataset.book) return `book:${dataset.library || ""}/${dataset.book}`;
  if (dataset.index !== undefined) return `index:${dataset.index}`;
  return "";
}

function activeContainer(document) {
  for (const selector of CONTAINERS) {
    const element = document.querySelector?.(selector);
    if (element && !element.hidden) return element;
  }
  return null;
}

/** The card under the sticky masthead, and its height relative to the viewport. */
export function captureCataloguePlace(document = globalThis.document, top = 0) {
  if (!document?.querySelector) return null;
  const container = activeContainer(document);
  if (!container?.querySelectorAll) return null;
  const card = [...container.querySelectorAll(CARDS)].find(
    (item) => item.getBoundingClientRect().bottom > top + 8,
  );
  const key = card && placeKey(card);
  if (!key) return null;
  return {
    container: container.id,
    key,
    top: card.getBoundingClientRect().top,
  };
}

/**
 * Scroll the remembered card back under the edge once it exists again. The
 * shelf loads asynchronously, so this waits a frame at a time — and gives up
 * quietly rather than scrolling to a guess.
 */
export function restoreCataloguePlace(
  place,
  {
    document = globalThis.document,
    window = globalThis.window,
    requestFrame = globalThis.requestAnimationFrame?.bind(globalThis) ||
      ((callback) => setTimeout(callback, 16)),
    frames = 240,
  } = {},
) {
  if (!place || !document?.querySelector || !window) return;
  let remaining = frames;
  const step = () => {
    const container = document.querySelector(`#${place.container}`);
    const card = [...(container?.querySelectorAll?.(CARDS) || [])].find(
      (item) => placeKey(item) === place.key,
    );
    if (card) {
      const delta = card.getBoundingClientRect().top - place.top;
      if (Math.abs(delta) >= 1) window.scrollTo(0, Math.max(0, window.scrollY + delta));
      return;
    }
    remaining -= 1;
    if (remaining > 0) requestFrame(step);
  };
  requestFrame(step);
}
