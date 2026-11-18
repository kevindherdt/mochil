// Read-aloud for EPUB 3 Media Overlays (e.g. Storyteller books). The reader
// library owns SMIL parsing, audio and sentence highlighting; this controller
// adds start-from-where-you-are, pause/resume, speed, and lock-screen controls.
export const readAloudRates = [0.8, 1, 1.15, 1.3, 1.5, 1.75, 2];

export function createReadAloud({ view, button, speedField, speedSelect, sleepField, sleepSelect, storage, mediaSession, getTitle, onError = () => {}, onSleep = () => {}, timers = { set: (fn, ms) => setTimeout(fn, ms), clear: (id) => clearTimeout(id) } }) {
  const overlay = view.mediaOverlay;
  if (!overlay) return null;

  let state = "idle";
  const savedRate = Number(storage?.getItem("bobobooks-readaloud-rate"));
  let rate = readAloudRates.includes(savedRate) ? savedRate : 1;
  overlay.setRate(rate);

  button.hidden = false;
  if (speedField) speedField.hidden = false;
  if (speedSelect) speedSelect.value = String(rate);
  if (sleepField) sleepField.hidden = false;

  function render() {
    button.setAttribute("aria-pressed", String(state === "playing"));
    button.setAttribute("aria-label", state === "playing" ? "Pause read aloud" : state === "paused" ? "Resume read aloud" : "Read aloud");
    button.dataset.state = state;
    if (mediaSession) mediaSession.playbackState = state === "playing" ? "playing" : state === "paused" ? "paused" : "none";
  }

  // Begin at the first sentence on, or after, the page being read.
  function startHere() {
    const location = view.lastLocation;
    const index = location?.index ?? 0;
    const contents = view.renderer?.getContents?.() || [];
    const doc = contents.find((item) => item.index === index)?.doc;
    const range = location?.range;
    const filter = (item) => {
      if (!doc || !range) return true;
      try {
        const element = doc.getElementById(item.text.split("#")[1]);
        return !element || range.comparePoint(element, 0) >= 0;
      } catch {
        return true;
      }
    };
    state = "playing";
    render();
    overlay.start(index, filter);
  }

  function toggle() {
    if (state === "idle") startHere();
    else if (state === "playing") {
      overlay.pause();
      state = "paused";
      render();
    } else {
      overlay.resume();
      state = "playing";
      render();
    }
  }

  function setRate(value) {
    if (!readAloudRates.includes(value)) return;
    rate = value;
    overlay.setRate(rate);
    storage?.setItem("bobobooks-readaloud-rate", String(rate));
  }

  // The sleep timer pauses reading after a set time, however far it has got.
  let sleepTimer = null;
  function setSleep(minutes) {
    if (sleepTimer) timers.clear(sleepTimer);
    sleepTimer = null;
    if (!(minutes > 0)) return;
    sleepTimer = timers.set(() => {
      sleepTimer = null;
      if (state === "playing") toggle();
      if (sleepSelect) sleepSelect.value = "0";
      onSleep();
    }, minutes * 60_000);
  }

  button.onclick = toggle;
  if (sleepSelect) sleepSelect.onchange = () => setSleep(Number(sleepSelect.value));
  if (speedSelect) speedSelect.onchange = () => setRate(Number(speedSelect.value));
  overlay.addEventListener("highlight", () => {
    if (state !== "playing") {
      state = "playing";
      render();
    }
  });
  overlay.addEventListener("error", (event) => {
    state = "paused";
    render();
    onError(event.detail);
  });

  if (mediaSession?.setActionHandler) {
    const handlers = {
      play: () => state !== "playing" && toggle(),
      pause: () => state === "playing" && toggle(),
      previoustrack: () => overlay.prev(),
      nexttrack: () => overlay.next(),
    };
    for (const [action, handler] of Object.entries(handlers)) {
      try { mediaSession.setActionHandler(action, handler); } catch { /* unsupported action */ }
    }
    if (typeof MediaMetadata !== "undefined") mediaSession.metadata = new MediaMetadata({ title: getTitle?.() || "Mochil" });
  }

  render();
  return { toggle, setRate, setSleep, get state() { return state; } };
}
