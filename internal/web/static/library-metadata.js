// Which online metadata provider a library uses. The server derives this from
// the library kind and the tokens saved under Metadata; the settings card only
// reports the same answer, so the two can never disagree.

const publicationKinds = ["books", "comics", "manga", "mixed"];
const videoKinds = ["movies", "tv", "anime"];

// What a library of this kind gets when nothing is saved for it.
export function derivedMetadataProvider(kind, providers = {}) {
  if (videoKinds.includes(kind))
    return providers?.tmdb_token_configured ? "TMDB" : "Off";
  if (publicationKinds.includes(kind))
    return providers?.hardcover_token_configured ? "Hardcover" : "Open Library";
  return "Off";
}

// What the library really uses: a saved name outranks the derivation while it
// still fits the kind and has its token, which is exactly what the server does.
// Reading only the saved field would claim Hardcover for a library that is
// looked up through Open Library, or "none" for one that is being looked up.
export function effectiveMetadataProvider(library, providers = {}) {
  const kind = String(library?.kind || "");
  const saved = String(library?.metadata_provider || "").trim().toLowerCase();
  if (saved === "off" || saved === "disabled") return "Off";
  if (
    saved === "hardcover" &&
    publicationKinds.includes(kind) &&
    providers?.hardcover_token_configured
  )
    return "Hardcover";
  if (saved === "openlibrary" && publicationKinds.includes(kind))
    return "Open Library";
  if (
    saved === "tmdb" &&
    videoKinds.includes(kind) &&
    providers?.tmdb_token_configured
  )
    return "TMDB";
  return derivedMetadataProvider(kind, providers);
}

export function metadataLineText(library, providers = {}) {
  return `Metadata: ${effectiveMetadataProvider(library, providers)}`;
}
