// Whether animations are off. A choice made in Settings wins; until then they are off with the E-ink
// theme and on a device that asks for less motion. Shared by the settings page and the reader.
export function motionReduced(storage, palette, deviceAsksForLess = false) {
  const choice = storage.getItem("bobobooks-reduce-motion");
  if (choice !== null) return choice === "1";
  return palette === "eink" || deviceAsksForLess;
}

export function applyMotion(root, storage, win = globalThis) {
  const palette = storage.getItem("bobobooks-palette") || "bobobooks";
  const asksForLess = Boolean(win.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches);
  root.dataset.motion = motionReduced(storage, palette, asksForLess) ? "reduced" : "full";
}
