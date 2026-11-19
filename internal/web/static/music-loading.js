export function musicLoadingHTML() {
  const rows = Array.from(
    { length: 8 },
    () =>
      '<span class="music-loading-row" aria-hidden="true"><i></i><b></b><small></small></span>',
  ).join("");
  return `<div class="music-loading" role="status"><span class="sr-only">Opening music</span>${rows}</div>`;
}
