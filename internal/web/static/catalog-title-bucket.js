/** Matches the server catalogue's single-article, A–Z/# bucket contract. */
export function catalogTitleBucket(value) {
  const normalized = String(value ?? '').trim().toLowerCase().replace(/^(the|an|a) /, '').trim()
  const first = normalized.charAt(0).toUpperCase()
  return /^[A-Z]$/.test(first) ? first : '#'
}
