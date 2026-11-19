// The A–Z rail stays out of the way until the page scrolls or the pointer
// approaches the right edge. It remains focusable and clickable at all times.
export function createAlphabetPeek({ document, window, delay = 1800 }) {
  const root = document.documentElement;
  let timer;
  const wake = () => {
    root.classList.add("rail-awake");
    clearTimeout(timer);
    timer = setTimeout(() => root.classList.remove("rail-awake"), delay);
  };
  window.addEventListener("scroll", wake, { passive: true });
  window.addEventListener(
    "pointermove",
    (event) => {
      if (event.clientX > window.innerWidth - 72) wake();
    },
    { passive: true },
  );
  return { wake };
}
