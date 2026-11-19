// The shared #status line is announced to assistive tech as before, but sighted
// readers get a quiet toast that fades on its own instead of a permanent row.
export function createStatusToast({ element, window, errorPattern = /could not|failed|error|expired|unavailable/i }) {
  let timer;
  const read = () => element.textContent.trim();
  function sync() {
    clearTimeout(timer);
    const text = read();
    element.classList.toggle("toast-visible", Boolean(text));
    if (!text) return;
    timer = window.setTimeout(
      () => element.classList.remove("toast-visible"),
      errorPattern.test(text) ? 12000 : 4500,
    );
  }
  new MutationObserver(sync).observe(element, { childList: true, characterData: true, subtree: true });
  sync();
  return { sync };
}
