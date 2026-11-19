// Where read-aloud starts. In order: the sentence the person selected, else the one narration stopped at (if it is on
// the page they are looking at), else the first sentence on the page. The place it stopped at is kept per book as the
// chapter and the opening words of the sentence, which still find it if the sentences are cut a little differently
// the next time the chapter opens.

const KEY_LENGTH = 60
export const sentenceKey = (text) => String(text || '').replace(/\s+/g, ' ').trim().slice(0, KEY_LENGTH)

export const placeOf = (href, sentence, index) => ({ href: String(href || '').split('#')[0], key: sentenceKey(sentence?.text), index })

// rememberedIndex finds the saved place among a chapter's sentences, or -1 when it is a different chapter or the text is gone.
export function rememberedIndex(sentences, place, href) {
  if (!place?.key || String(href || '').split('#')[0] !== place.href) return -1
  let best = -1;
  sentences.forEach((sentence, index) => {
    if (sentenceKey(sentence.text) === place.key && (best < 0 || Math.abs(index - place.index) < Math.abs(best - place.index))) best = index;
  })
  return best
}

// startIndex picks the sentence to begin with. `visibleAt(i)` says whether sentence i is on the page being looked at, and
// `first` is the first sentence on that page.
export function startIndex({ selected = -1, remembered = -1, visibleAt, first }) {
  if (selected >= 0) return selected
  if (remembered >= 0 && visibleAt(remembered)) return remembered
  return first
}

export function readPlace(storage, name) {
  try {
    const place = JSON.parse(storage.getItem(name) || 'null')
    return place && typeof place.href === 'string' && typeof place.key === 'string' ? place : null
  } catch { return null }
}

export function savePlace(storage, name, place) {
  try { storage.setItem(name, JSON.stringify(place)) } catch {}
}
