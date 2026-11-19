// A long table only needs the rows that are on screen. Every row is kept in memory, but only the ones in view (plus a few
// either side) are drawn, and blank space above and below stands in for the rest, so the scrollbar is the length of the
// whole list and scrolling to the end is one drag. This is how a desktop program shows twenty thousand rows.

export const OVERSCAN = 12;

// windowFor says which rows to draw for a box scrolled `scrollTop` pixels down and `viewport` pixels tall, and how much
// blank space to leave above and below them.
export function windowFor({ scrollTop, viewport, rowHeight, count, overscan = OVERSCAN }) {
  const height = Math.max(1, rowHeight);
  const start = Math.max(0, Math.floor(Math.max(0, scrollTop) / height) - overscan);
  const end = Math.min(count, Math.ceil((Math.max(0, scrollTop) + viewport) / height) + overscan);
  return { start, end: Math.max(start, end), before: start * height, after: Math.max(0, (count - Math.max(start, end)) * height) };
}

// stillCovers says whether the rows already drawn still cover what is on screen with some to spare, so a small scroll
// does not redraw anything.
export function stillCovers(drawn, { scrollTop, viewport, rowHeight, count }, spare = 3) {
  const first = Math.floor(Math.max(0, scrollTop) / Math.max(1, rowHeight));
  const last = Math.ceil((Math.max(0, scrollTop) + viewport) / Math.max(1, rowHeight));
  const startOk = drawn.start === 0 || first >= drawn.start + spare;
  const endOk = drawn.end >= count || last <= drawn.end - spare;
  return startOk && endOk;
}

// scrollTopFor is where to scroll so that row `index` is fully visible below a pinned heading `header` pixels tall,
// moving as little as possible; it returns the current position if the row is already visible.
export function scrollTopFor(index, { scrollTop, viewport, rowHeight, header = 0 }) {
  const top = index * rowHeight;
  if (top < scrollTop) return top;
  if (top + rowHeight > scrollTop + viewport - header) return top + rowHeight - viewport + header;
  return scrollTop;
}
