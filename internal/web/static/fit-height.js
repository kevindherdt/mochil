// The editor's table scrolls inside its own box, so its horizontal scrollbar and its pinned column headings are always on
// screen. Without this the box grows with its rows, and both end up far below the window.

// heightFor is the tallest the box may be so that its bottom edge stays inside a window of `viewport` pixels, given
// where its top edge is on the page. It never gets smaller than `minimum`, so a short window still shows some rows.
export function heightFor(top, viewport, { margin = 16, minimum = 240 } = {}) {
  return Math.max(minimum, Math.round(viewport - top - margin));
}

// fitHeight keeps `element` that tall as the window is resized, and returns a function to call when what is above the
// box has changed size (a selection bar appearing, filters wrapping).
export function fitHeight(element, view = window, options) {
  const apply = () => { element.style.maxHeight = `${heightFor(element.getBoundingClientRect().top + (view.scrollY || 0), view.innerHeight, options)}px`; };
  view.addEventListener("resize", apply);
  return apply;
}
