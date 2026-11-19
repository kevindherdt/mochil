// A drop-down list anchored to its button can run off the window when the button wraps to a different spot on a narrow
// screen. When it opens, measure it and slide it back inside, with a small margin.
const MARGIN = 8;

// shiftToFit says how far (in pixels, negative is left) a box from `left` to `right` must move to sit inside `width`.
export function shiftToFit(left, right, width, margin = MARGIN) {
  if (right - left > width - 2 * margin) return margin - left; // wider than the window: start at the left edge
  if (left < margin) return margin - left;
  if (right > width - margin) return width - margin - right;
  return 0;
}

export function keepInView(details, list, view = window) {
  details.addEventListener("toggle", () => {
    list.style.transform = "";
    if (!details.open) return;
    const box = list.getBoundingClientRect();
    const shift = shiftToFit(box.left, box.right, view.innerWidth);
    if (shift) list.style.transform = `translateX(${Math.round(shift)}px)`;
  });
}
