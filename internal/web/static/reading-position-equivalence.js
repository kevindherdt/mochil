const fractionEquivalent = (left, right) => Number.isFinite(Number(left)) && Number.isFinite(Number(right)) && Math.abs(Number(left) - Number(right)) <= 0.002

function fixedPage(position) {
  for (const kind of ['comic', 'pdf']) {
    if (!(kind in (position?.locators || {}))) continue
    try {
      const page = JSON.parse(position.locators[kind])?.page
      return { present: true, valid: Number.isInteger(page) && page >= 0, page }
    } catch { return { present: true, valid: false, page: null } }
  }
  return { present: false, valid: false, page: null }
}

function sameAnchor(left, right) {
  if (!left || !right) return false
  const leftSpine = left.spine_index ?? left.spineIndex
  const rightSpine = right.spine_index ?? right.spineIndex
  const leftExact = left.exact || left.text || ''
  const rightExact = right.exact || right.text || ''
  if (!Number.isInteger(leftSpine) || leftSpine !== rightSpine || !leftExact || leftExact !== rightExact) return false
  if ((!left.prefix && !left.suffix) || (!right.prefix && !right.suffix)) return false
  return (left.prefix || '') === (right.prefix || '') && (left.suffix || '') === (right.suffix || '')
}

export function readingPositionsEquivalent(left, right) {
  const leftPage = fixedPage(left)
  const rightPage = fixedPage(right)
  if (leftPage.present || rightPage.present) {
    return leftPage.present && rightPage.present && leftPage.valid && rightPage.valid && leftPage.page === rightPage.page
  }
  if (fractionEquivalent(left?.fraction, right?.fraction)) return true
  if (left?.cfi && left.cfi === right?.cfi) return true
  if (left?.koreader_locator && left.koreader_locator === right?.koreader_locator) return true
  return sameAnchor(left?.web_anchor, right?.web_anchor)
}

