const languageAliases = {
  en: "eng",
  english: "eng",
  ja: "jpn",
  japanese: "jpn",
  ko: "kor",
  korean: "kor",
  nl: "nld",
  dutch: "nld",
  fr: "fra",
  french: "fra",
  de: "deu",
  german: "deu",
  es: "spa",
  spanish: "spa",
  zh: "zho",
  chinese: "zho",
};

export function videoLanguageCode(value) {
  const normalized = String(value || "")
    .trim()
    .toLowerCase();
  return languageAliases[normalized] || normalized;
}

export function preferredVideoTrack(tracks, language) {
  return tracks.find(
    (track) =>
      videoLanguageCode(track.language) === videoLanguageCode(language),
  );
}

export function videoTrackLabel(track, fallback) {
  return (
    [track.title, track.language?.toUpperCase(), track.codec?.toUpperCase()]
      .filter(Boolean)
      .join(" · ") || fallback
  );
}

export function videoTimeLabel(seconds) {
  if (!Number.isFinite(seconds) || seconds <= 0) return "";
  const rounded = Math.floor(seconds);
  const hours = Math.floor(rounded / 3600);
  const minutes = Math.floor((rounded % 3600) / 60);
  const remainder = rounded % 60;
  return hours
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`
    : `${minutes}:${String(remainder).padStart(2, "0")}`;
}

const releaseTag = /\s+(?:(?:blu-?ray|bdrip|brrip|web-?dl|web-?rip|hdtv|dvdrip|remux)\b|(?:480|576|720|1080|2160)p\b).*$/i;

/** A movie title for display: file names carry release tags ("… (2018) Bluray-1080p") that mean nothing in a poster grid. */
export function cleanVideoTitle(title) {
  const text = String(title || "");
  return text.replace(releaseTag, "").trim() || text;
}
