// Release names are file names, not titles. "When They Cry Higurashi (2006) - S01E05 - 005 - The Cotton
// Drifting Chapter Part 1 Jealousy [Bluray-576p v2][Opus 2 0][JA][x265 10bit]-Okay-Subs" is unreadable on a
// card, and every one of those trailing groups is information the player already knows from the file itself.
const TECHNICAL = /(2160p|1080p|720p|480p|blu-?ray|brrip|bdrip|web-?dl|webrip|hdtv|dvdrip|x26[45]|hevc|av1|opus|aac|ddp?5|ac3|flac|10bit|8bit|hdr|subs?|dubs?|multi|dual)/i

export function cleanVideoTitle(raw, { episode = 0, season = 0 } = {}) {
  let title = String(raw || '').trim()
  if (!title) return ''

  // Bracketed groups go entirely; parenthesised ones only when they read as release notes, so "(2006)" stays.
  title = title.replace(/\[[^\]]*\]/g, ' ')
  title = title.replace(/\(([^)]*)\)/g, (whole, inner) => (TECHNICAL.test(inner) ? ' ' : whole))
  // A trailing "-GROUP" (release team) is never part of a title.
  title = title.replace(/[-_.\s]+[A-Za-z0-9]+$/g, (whole) => (/\b(x265|x264|subs?|group)\b/i.test(whole) ? ' ' : whole))
  title = title.replace(/[-_.\s]*-(okay|subs|sub|group|rarbg|yts|eztv)\b.*$/i, ' ')
  // Scene separators: "Show.Name.S01E05" -> "Show Name S01E05".
  title = title.replace(/[._]+/g, ' ')
  // S01E05 -> S1 E5, and 1x05 stays as it is.
  title = title.replace(/\bS(\d{1,2})[\s._-]*E(\d{1,3})\b/gi, (_, s, e) => `S${Number(s)} E${Number(e)}`)

  // A second, bare episode number after the marker ("- 005 -") repeats what the marker already said.
  if (episode > 0 || /S\d{1,2} E\d{1,3}/i.test(title)) {
    title = title.replace(/\s+-\s+\d{1,4}\s+-\s+/g, ' · ')
  }

  // Scene names separate with dots, so technical tokens end up as words: drop them from the end, where they sit.
  const tokens = title.trim().split(/\s+/)
  while (tokens.length > 1 && (TECHNICAL.test(tokens[tokens.length - 1]) || /^(group|subs?|dub|multi)$/i.test(tokens[tokens.length - 1]) || /^[A-Z0-9]{2,}$/.test(tokens[tokens.length - 1]) && tokens.length > 3 && /x26[45]|hevc/i.test(tokens[tokens.length - 2] || ''))) {
    tokens.pop()
  }
  title = tokens.join(' ')

  title = title.replace(/\s*[-–—]\s*$/g, ' ').replace(/\s{2,}/g, ' ').replace(/^[\s·-]+|[\s·-]+$/g, '')
  return title.trim() || String(raw || '').trim()
}
