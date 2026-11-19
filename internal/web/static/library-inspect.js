// Turning a chosen folder into a library card. The server reads the folder and
// answers with a suggestion; this module turns that answer into the card the
// settings page renders, and keeps a usable fallback when the answer is missing.
import { defaultPreferredFormats, newLibrary } from "./library-form-model.js?v=1";

const libraryKinds = [
  "books",
  "comics",
  "manga",
  "mixed",
  "movies",
  "tv",
  "anime",
  "audiobooks",
  "podcasts",
  "music",
  "documents",
  "vault",
];

// A folder whose contents could not be read still needs a name the reader
// recognises; the last segment of the path they picked is the neutral choice.
export function libraryNameFromPath(path) {
  const trimmed = String(path || "").replace(/[\\/]+$/, "");
  const segment = trimmed.split(/[\\/]/).pop() || "";
  return segment.trim() || "New library";
}

export function inspectionEvidence(inspection) {
  return (Array.isArray(inspection?.evidence) ? inspection.evidence : [])
    .filter((item) => typeof item === "string" && item.trim())
    .map((item) => item.trim())
    .join(" · ");
}

// Builds the card from GET /api/library-inspect. Its answer is a suggestion, not
// a gate: a failed or empty response still leaves a usable card with the chosen
// folder and a name taken from the path.
export function libraryFromInspection(path, inspection, number = 1) {
  const suggestion = inspection?.suggested || {};
  const kind = libraryKinds.includes(suggestion.kind) ? suggestion.kind : "books";
  const library = newLibrary(number);
  library.name = String(suggestion.name || "").trim() || libraryNameFromPath(path);
  library.kind = kind;
  library.source = {
    type:
      suggestion.storage === "calibre" &&
      !["documents", "vault"].includes(kind)
        ? "calibre"
        : "filesystem",
    path,
  };
  library.preferred_formats = defaultPreferredFormats(kind);
  if (kind === "manga") library.inference.folder_layout = "series";
  return library;
}

// Detection is best effort. Anything the endpoint cannot tell us — a failure, an
// empty body or no suggestion at all — falls back to the card built from the path
// alone, so choosing a folder always produces something the reader can tweak.
export async function inspectFolderLibrary(path, { requestJSON, number = 1 } = {}) {
  let inspection = null;
  try {
    if (requestJSON)
      inspection =
        (await requestJSON(
          `/api/library-inspect?path=${encodeURIComponent(path)}`,
        )).data || null;
  } catch {
    inspection = null;
  }
  return {
    library: libraryFromInspection(path, inspection, number),
    evidence: inspectionEvidence(inspection),
  };
}
