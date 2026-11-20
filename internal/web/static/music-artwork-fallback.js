/** Deterministic, language-safe fallback for music without usable artwork. */
export function musicArtworkMonogram(value) {
  const words = String(value || '').match(/[\p{L}\p{N}]+/gu) || []
  if (!words.length) return '♪'
  const letters = words.length === 1
    ? [...words[0]].slice(0, 2)
    : [words[0][0], words.at(-1)[0]]
  return letters.join('').toLocaleUpperCase()
}
