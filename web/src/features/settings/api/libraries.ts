// features/settings/api/libraries.ts — configuring libraries: the config entries inside the settings document, the
// folder browser that picks a path, and the manual scan.
//
// What the server actually does on save matters for how this is written. `PUT /api/settings` is a *live* apply
// (cmd/mochil/live.go's `apply`): the new config is written to a pending file, changed libraries are opened and
// scanned, the handler is swapped, and the response reports what happened. Adding a library therefore takes effect
// immediately — no restart, no "scan later" step. That is why the panel reports `libraries_scanned` instead of
// telling the reader to go and press something else.
//
// Existing library objects are never rebuilt: adding appends a new object and removing filters one out, so every
// field this app has never heard of (preferred formats, per-library scan interval, calibre options, vault writes)
// survives untouched. Only the object this app *creates* is written from scratch, and it is written with the same
// field set the Go struct marshals (internal/appconfig/config.go's `Library`), including empty strings for the
// source fields a filesystem library doesn't use.
import { z } from "zod";
import { fetchSettings, putSettings, type SettingsApplyResult, type SettingsDocument } from "./settings";
import { request } from "../../../lib/http";

export type LibraryKind =
  | "books"
  | "comics"
  | "manga"
  | "mixed"
  | "movies"
  | "tv"
  | "anime"
  | "audiobooks"
  | "podcasts"
  | "music"
  | "documents"
  | "vault";

/** The server's own accepted values (appconfig's validation): anything else is rejected on save. */
export const LIBRARY_KINDS: { id: LibraryKind; label: string }[] = [
  { id: "books", label: "Books" },
  { id: "comics", label: "Comics" },
  { id: "manga", label: "Manga" },
  { id: "mixed", label: "Mixed" },
  { id: "audiobooks", label: "Audiobooks" },
  { id: "podcasts", label: "Podcasts" },
  { id: "music", label: "Music" },
  { id: "movies", label: "Movies" },
  { id: "tv", label: "TV" },
  { id: "anime", label: "Anime" },
  { id: "documents", label: "Documents" },
  { id: "vault", label: "Obsidian vault" },
];

export interface LibraryEntry {
  id: string;
  name: string;
  kind: string;
  /** `source.path` — the folder or Calibre directory this library reads. Empty for a remote source. */
  path: string;
  /** The raw object, so a caller can inspect fields this app doesn't model without a second parse. */
  raw: Record<string, unknown>;
}

/** Reads the `libraries` array defensively: a field that isn't the type expected reads as empty rather than
 *  throwing, because one malformed entry must not make the whole panel — or the whole settings document — unusable. */
export function readLibraryEntries(settings: SettingsDocument): LibraryEntry[] {
  const libraries = settings.libraries;
  if (!Array.isArray(libraries)) return [];
  const entries: LibraryEntry[] = [];
  for (const value of libraries) {
    if (typeof value !== "object" || value === null || Array.isArray(value)) continue;
    const raw = value as Record<string, unknown>;
    const source = typeof raw.source === "object" && raw.source !== null ? (raw.source as Record<string, unknown>) : {};
    entries.push({
      id: typeof raw.id === "string" ? raw.id : "",
      name: typeof raw.name === "string" ? raw.name : "",
      kind: typeof raw.kind === "string" ? raw.kind : "",
      path: typeof source.path === "string" ? source.path : "",
      raw,
    });
  }
  return entries;
}

/** A library id the server will accept: `^[a-z0-9][a-z0-9-]{0,62}$`, derived from the name and made unique. */
export function libraryIdFor(name: string, existing: string[]): string {
  const base =
    name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 48) || "library";
  const startsValid = /^[a-z0-9]/.test(base) ? base : `l-${base}`;
  if (!existing.includes(startsValid)) return startsValid;
  for (let suffix = 2; suffix < 1000; suffix += 1) {
    const candidate = `${startsValid}-${suffix}`;
    if (!existing.includes(candidate)) return candidate;
  }
  return `${startsValid}-${Date.now()}`;
}

/** GET /api/library-inspect?path=… — what a folder looks like before it becomes a library: the name to suggest, the
 *  kind its files imply, whether it is a Calibre library (a `metadata.db` at the top), and the evidence behind it.
 *  This is what lets "pick a folder" be the whole add flow instead of asking someone to classify their own files. */
export interface LibraryInspection {
  path: string;
  suggestedName: string;
  suggestedKind: LibraryKind;
  /** "filesystem" or "calibre" — the shape `source.type` must be written with. */
  storage: string;
  evidence: string[];
}

const inspectionSchema = z.looseObject({
  path: z.string().optional(),
  suggested: z
    .looseObject({ name: z.string().optional(), kind: z.string().optional(), storage: z.string().optional() })
    .optional(),
  evidence: z.array(z.string()).nullable().optional(),
});

export async function inspectLibraryFolder(path: string): Promise<LibraryInspection> {
  const raw = await request<unknown>(`/api/library-inspect?path=${encodeURIComponent(path)}`);
  const parsed = inspectionSchema.parse(raw);
  const suggested = parsed.suggested ?? {};
  const kind = (LIBRARY_KINDS.find((entry) => entry.id === suggested.kind)?.id ?? "books") as LibraryKind;
  return {
    path: parsed.path ?? path,
    suggestedName: suggested.name ?? "",
    suggestedKind: kind,
    storage: suggested.storage === "calibre" ? "calibre" : "filesystem",
    evidence: parsed.evidence ?? [],
  };
}

export interface NewLibrary {
  name: string;
  kind: LibraryKind;
  path: string;
  /** "calibre" when the folder holds a `metadata.db`; the server reads those differently from a plain folder. */
  storage?: string;
}

/** Adds one library and returns the server's apply report (which says how many libraries it opened and scanned). */
export async function addLibrary(input: NewLibrary): Promise<SettingsApplyResult> {
  const document = await fetchSettings();
  const existing = readLibraryEntries(document);
  const library = {
    id: libraryIdFor(input.name, existing.map((entry) => entry.id)),
    name: input.name,
    kind: input.kind,
    // Left empty on purpose: the server derives the provider from the kind (appconfig's
    // `automaticMetadataProvider` — Hardcover/Open Library for publications, TMDB for video, nothing else).
    metadata_provider: "",
    page_streaming: false,
    source: {
      type: input.storage === "calibre" ? "calibre" : "filesystem",
      path: input.path,
      url: "",
      repository: "",
      repository_name: "",
      root: "",
      manifest: "",
      token_file: "",
    },
    // "auto" lets the scanner work the layout out from the folders rather than being told.
    inference: { folder_layout: "auto", filename_author: false },
  };
  return putSettings({ ...document, libraries: [...(Array.isArray(document.libraries) ? document.libraries : []), library] });
}

/** Removes a library from the configuration. The files are not touched — only what Mochil reads changes. */
export async function removeLibrary(id: string): Promise<SettingsApplyResult> {
  const document = await fetchSettings();
  const remaining = (Array.isArray(document.libraries) ? document.libraries : []).filter((value) => {
    if (typeof value !== "object" || value === null) return true;
    return (value as Record<string, unknown>).id !== id;
  });
  return putSettings({ ...document, libraries: remaining });
}

/** Renames a library, or changes its kind, by editing its entry in the settings document in place.
 *
 *  Kind is the one field worth a warning: it decides which scanner reads the folder *and* which metadata provider
 *  the server derives (appconfig's `automaticMetadataProvider`), so changing "books" to "music" re-reads the same
 *  files a different way. Nothing is deleted — the index is rebuilt on the next scan — but the caller is expected to
 *  say so before offering the control, rather than this function guessing.
 *
 *  Every other field on the entry is preserved as-is: this is an edit, not a rewrite, and `page_streaming`,
 *  `source`, `inference` and the rest are the server's to keep. */
/** The per-library switches the legacy settings page offered and this panel used to leave alone. Every one of them is
 *  validated by the server (internal/appconfig/config.go) — an unsupported format, a duplicate, or metadata writes on a
 *  non-filesystem source is a rejected save, so the options offered here are exactly the accepted ones. */
export interface LibraryOptions {
  /** Empty means "same as the server default"; "0" means only when scanned by hand. */
  scanEvery: string;
  pageStreaming: boolean;
  /** Only for a filesystem or Calibre source on a publication library — the server rejects everything else. */
  metadataWrites: boolean;
  /** Order of preference when one book exists in several formats. Empty means the server's default for the kind. */
  preferredFormats: string[];
}

export const SCAN_INTERVAL_OPTIONS: { id: string; label: string }[] = [
  { id: "", label: "Same as the server default" },
  { id: "5m", label: "Every 5 minutes" },
  { id: "30m", label: "Every 30 minutes" },
  { id: "1h", label: "Every hour" },
  { id: "6h", label: "Every 6 hours" },
  { id: "24h", label: "Once a day" },
  { id: "0", label: "Only when I scan it" },
];

/** The five formats the server accepts, in the order a rip is usually preferred. */
export const PREFERRED_FORMATS = ["epub", "pdf", "mobi", "azw3", "cbz"];

/** Publication kinds are the ones where a book can exist in several formats and where metadata lives in the file. */
export function isPublicationKind(kind: string): boolean {
  return kind === "books" || kind === "comics" || kind === "manga" || kind === "mixed";
}

/** Whether the source type lets Mochil write metadata back into the files. */
export function supportsMetadataWrites(entry: LibraryEntry): boolean {
  const source = entry.raw.source;
  const type = typeof source === "object" && source !== null ? (source as Record<string, unknown>).type : undefined;
  return type === "filesystem" || type === "calibre";
}

export function readLibraryOptions(entry: LibraryEntry): LibraryOptions {
  const raw = entry.raw;
  return {
    scanEvery: typeof raw.scan_every === "string" ? raw.scan_every : "",
    pageStreaming: raw.page_streaming === true,
    metadataWrites: raw.metadata_writes === true,
    preferredFormats: Array.isArray(raw.preferred_formats)
      ? raw.preferred_formats.filter((value): value is string => typeof value === "string")
      : [],
  };
}

export async function updateLibrary(
  id: string,
  patch: { name?: string; kind?: string; options?: LibraryOptions },
): Promise<SettingsApplyResult> {
  const document = await fetchSettings();
  const libraries = Array.isArray(document.libraries) ? document.libraries : [];
  let found = false;
  const next = libraries.map((value) => {
    if (typeof value !== "object" || value === null) return value;
    const entry = value as Record<string, unknown>;
    if (entry.id !== id) return value;
    found = true;
    const options = patch.options;
    return {
      ...entry,
      ...(patch.name === undefined ? {} : { name: patch.name }),
      ...(patch.kind === undefined ? {} : { kind: patch.kind }),
      ...(options === undefined
        ? {}
        : {
            scan_every: options.scanEvery,
            page_streaming: options.pageStreaming,
            metadata_writes: options.metadataWrites,
            // Omitted rather than empty so the server keeps applying its own per-kind default.
            ...(options.preferredFormats.length === 0 ? {} : { preferred_formats: options.preferredFormats }),
          }),
    };
  });
  if (!found) throw new Error("That library is no longer in the configuration — reload the page.");
  return putSettings({ ...document, libraries: next });
}

/** POST /api/library-scan — rescan one library now. 202 `{job_keys, status: "queued"}`; the scan itself runs in the
 *  background, so this returns as soon as it is scheduled. */
export async function scanLibrary(libraryId: string): Promise<void> {
  await request<unknown>("/api/library-scan", { method: "POST", body: { library_id: libraryId } });
}

export interface FolderListing {
  path: string;
  parent: string | undefined;
  directories: { name: string; path: string }[];
  truncated: boolean;
}

/** GET /api/folders — the configured browsable roots, or the sub-folders of one path. */
export async function fetchFolders(path: string): Promise<FolderListing> {
  const query = path === "" ? "" : `?path=${encodeURIComponent(path)}`;
  const raw = await request<unknown>(`/api/folders${query}`);
  const listing = (typeof raw === "object" && raw !== null ? raw : {}) as Partial<FolderListing>;
  return {
    path: typeof listing.path === "string" ? listing.path : "",
    parent: typeof listing.parent === "string" ? listing.parent : undefined,
    directories: Array.isArray(listing.directories)
      ? listing.directories.filter((entry): entry is { name: string; path: string } =>
          typeof entry === "object" && entry !== null && typeof entry.name === "string" && typeof entry.path === "string",
        )
      : [],
    truncated: listing.truncated === true,
  };
}
