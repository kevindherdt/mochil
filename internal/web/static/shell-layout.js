export function createShellLayout({
  window,
  root,
  body,
  masthead,
  tools,
  libraryView,
  onScroll,
  ResizeObserver,
  requestFrame = requestAnimationFrame,
}) {
  function updateStickyMetrics() {
    const mastheadHeight = masthead.offsetHeight;
    const toolsHeight = 0;
    const mastheadValue = `${mastheadHeight}px`;
    const stickyValue = `${mastheadHeight + toolsHeight}px`;
    if (root.style.getPropertyValue("--masthead-height") !== mastheadValue)
      root.style.setProperty("--masthead-height", mastheadValue);
    if (root.style.getPropertyValue("--catalog-sticky-top") !== stickyValue)
      root.style.setProperty("--catalog-sticky-top", stickyValue);
  }
  let frame = 0;
  function scheduleScrollState() {
    if (frame) return;
    frame = requestFrame(() => {
      frame = 0;
      body.classList.toggle("shell-scrolled", window.scrollY > 18);
      onScroll?.(
        Number.parseFloat(
          root.style.getPropertyValue("--catalog-sticky-top"),
        ) || 0,
      );
    });
  }
  window.addEventListener("scroll", scheduleScrollState, { passive: true });
  window.addEventListener("resize", updateStickyMetrics);
  new ResizeObserver(updateStickyMetrics).observe(masthead);
  new ResizeObserver(updateStickyMetrics).observe(tools);
  requestFrame(() => {
    updateStickyMetrics();
    scheduleScrollState();
  });
  return { updateStickyMetrics, scheduleScrollState };
}
