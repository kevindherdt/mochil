// Which part of the page turns it when tapped. "edges" is the outer quarters, "one hand" is the left fifth
// back, a strip in the middle for the menu and everything else forward, "off" never turns the page. The middle always shows or hides the menu.
export const tapZoneChoices = ["edges", "one hand", "off"];

export function tapAction(zones, x, width, rightToLeft = false) {
  if (zones === "off" || !(width > 0)) return "menu";
  const share = Math.min(1, Math.max(0, x / width));
  let side;
  if (zones === "one hand") side = share < 0.2 ? "back" : share >= 0.4 && share <= 0.6 ? "menu" : "forward";
  else side = share < 0.25 ? "back" : share > 0.75 ? "forward" : "menu";
  if (!rightToLeft || side === "menu") return side;
  return side === "back" ? "forward" : "back";
}

export function savedTapZones(storage) {
  try {
    const saved = storage.getItem("bobobooks-tap-zones");
    return tapZoneChoices.includes(saved) ? saved : "edges";
  } catch { return "edges"; }
}
