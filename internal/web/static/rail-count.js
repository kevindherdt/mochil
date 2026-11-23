// The design puts the item count on the right of every rail header ("Continue Watching & Listening   4 items").
// The rails render in three different modules, so the counting lives here rather than three times over.
export function railCountText(count) {
  const n = Number(count) || 0
  if (n <= 0) return ''
  return `${n} ${n === 1 ? 'item' : 'items'}`
}

export function setRailCount(document, id, count) {
  const node = document?.getElementById?.(id)
  if (!node) return
  node.textContent = railCountText(count)
}
