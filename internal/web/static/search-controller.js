export function createSearchController({
  masthead,
  panel,
  toggle,
  input,
  label,
  onSearch,
  requestFrame = requestAnimationFrame,
  delay = 250,
}) {
  let timer;
  // What the shelf is filtered by — not just what the input happens to say.
  let applied = input.value ?? "";

  function settle() {
    clearTimeout(timer);
    timer = undefined;
    applied = input.value;
    onSearch();
  }

  // Closing always drops the pending keystroke, so a query typed and dismissed
  // inside the debounce window cannot replace the shelf behind a closed panel.
  // Dismissing (not submitting) also puts back the unfiltered shelf when a
  // query is already applied, since the person can no longer see the reason
  // for the filtered results.
  function setOpen(open, { restore = false } = {}) {
    const wasOpen = toggle.getAttribute("aria-expanded") === "true";
    if (open === wasOpen) return;
    if (!open) {
      clearTimeout(timer);
      timer = undefined;
      if (restore && applied) {
        applied = "";
        input.value = "";
        onSearch();
      }
    }
    masthead.classList.toggle("searching", open);
    panel.inert = !open;
    panel.setAttribute("aria-hidden", String(!open));
    toggle.setAttribute("aria-expanded", String(open));
    toggle.setAttribute("aria-label", open ? "Close search" : label());
    if (open) requestFrame(() => input.focus());
  }

  toggle.onclick = () => {
    if (toggle.getAttribute("aria-expanded") === "true")
      setOpen(false, { restore: true });
    else setOpen(true);
  };
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      setOpen(false, { restore: true });
      toggle.focus();
      return;
    }
    if (event.key !== "Enter" || event.isComposing) return;
    event.preventDefault();
    settle();
    setOpen(false);
  });
  input.oninput = () => {
    clearTimeout(timer);
    timer = setTimeout(settle, delay);
  };
  return { setOpen };
}
