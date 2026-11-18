// What is not worth reading aloud. Books carry pages that are made for the eye:
// copyright notices, catalogue records, web addresses, advertisements for other
// books. Listening to them is the first thing that makes narration tiresome.

const NOISE = [
  /https?:\/\/|\bwww\.|\b[\w.+-]+@[\w-]+\.[a-z]{2,}\b|\b[\w-]+\.(?:com|net|org|co\.uk|com\.au)\b/i,
  /©|\(c\)\s*\d{4}|\bISBN\b|\ball rights reserved\b|\bLibrary of Congress\b|\bCataloging-in-Publication\b|^(?:printed in|first (?:published|edition|ebook edition)|cover (?:design|illustration|art)|ebook edition|source isbn)\b/i,
  /\b(?:newsletter|sign up for|discover the world|don['’]t miss)\b/i,
]

/** False for a sentence that should be skipped. */
export function worthReading(text) {
  return !NOISE.some(pattern => pattern.test(text))
}

const MATTER_TITLE = /^(?:copyright(?: page)?|title page|half title|contents|table of contents|also by[\w\s.,'’-]*|books by[\w\s.,'’-]*|about the (?:author|publisher)s?|about the translator|acknowledg(?:e)?ments?|praise(?: for[\w\s.,'’:-]*)?|newsletter|discover more[\w\s.,'’-]*|colophon|credits|permissions|bibliography|index|excerpt|preview|sneak peek|read on|coming soon|ebook extras?)\s*$/i

/** True for a chapter that is front or back matter, going by its title in the table of contents. */
export function isMatterTitle(title) {
  return MATTER_TITLE.test(String(title || '').replace(/\s+/g, ' ').trim())
}
