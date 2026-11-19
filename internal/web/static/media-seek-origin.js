/** Distinguishes player restoration from a deliberate browser-media seek. */
export function createMediaSeekOrigin(toleranceSeconds = 1) {
  let expected = null

  function expectAutomatic(positionSeconds) {
    const position = Number(positionSeconds)
    expected = Number.isFinite(position) ? Math.max(0, position) : null
  }

  function isExplicit(positionSeconds) {
    const position = Number(positionSeconds)
    const automatic = Number.isFinite(position) && Number.isFinite(expected) &&
      Math.abs(position - expected) <= toleranceSeconds
    expected = null
    return !automatic
  }

  function clear() { expected = null }

  return { clear, expectAutomatic, isExplicit }
}
