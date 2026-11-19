import assert from "node:assert/strict";
import test from "node:test";
import {
  applyLibraryKind,
  defaultPreferredFormats,
  derivedMetadataProvider,
  inspectFolderLibrary,
  inspectionEvidence,
  libraryFromInspection,
  libraryIdFromName,
  libraryNameFromPath,
  metadataLineText,
  newLibrary,
  normalizePreferredFormats,
  shouldShowFirstLibraryHint,
} from "../static/library-settings.js";

test("new libraries receive independent safe publication defaults", () => {
  const first = newLibrary(1);
  const second = newLibrary(2);
  assert.equal(first.id, "new-library-1");
  assert.equal(first.source.type, "filesystem");
  assert.equal(first.source.path, "");
  assert.deepEqual(first.preferred_formats, ["epub", "pdf", "mobi", "azw3"]);
  first.source.path = "/changed";
  first.preferred_formats.push("txt");
  assert.equal(second.source.path, "");
  assert.deepEqual(second.preferred_formats, defaultPreferredFormats("books"));
});

test("preferred formats normalize extensions and discard empty entries", () => {
  assert.deepEqual(normalizePreferredFormats(" .EPUB, pdf, , .CBZ "), [
    "epub",
    "pdf",
    "cbz",
  ]);
  assert.deepEqual(normalizePreferredFormats(""), []);
});

test("kind changes migrate only untouched defaults and enforce compatible storage", () => {
  const manga = newLibrary(1);
  applyLibraryKind(manga, "manga");
  assert.deepEqual(manga.preferred_formats, ["cbz", "pdf", "epub"]);
  assert.equal(manga.inference.folder_layout, "series");

  const documents = newLibrary(2);
  documents.source.type = "calibre";
  documents.preferred_formats = ["custom"];
  applyLibraryKind(documents, "documents");
  assert.equal(documents.source.type, "filesystem");
  assert.deepEqual(documents.preferred_formats, ["custom"]);
});

// Changing Content used to be able to strand a saved provider that no longer
// fitted the kind. The select that cleared it is gone, and the whole settings
// save is validated, so leaving that value behind made every later save fail
// with an error the UI could not explain or undo.
test("changing the content drops a saved provider instead of stranding it", () => {
  const library = newLibrary(1);
  library.metadata_provider = "hardcover";
  applyLibraryKind(library, "audiobooks");
  assert.equal("metadata_provider" in library, false);

  const video = newLibrary(2);
  video.metadata_provider = "tmdb";
  applyLibraryKind(video, "books");
  assert.equal("metadata_provider" in video, false);
});

test("library ids are generated from names and stay unique", () => {
  assert.equal(libraryIdFromName("Light Novels"), "light-novels");
  assert.equal(libraryIdFromName("  Café / Bücher! "), "cafe-bucher");
  assert.equal(libraryIdFromName("日本語"), "library");
  assert.equal(libraryIdFromName("Books", new Set(["books", "books-2"])), "books-3");
  assert.match(libraryIdFromName("x".repeat(200)), /^[a-z0-9][a-z0-9-]{0,62}$/);
});

// The first install should say what to do in plain words, and only when there is nothing yet.
test("the first-library hint shows only while no library exists", () => {
  assert.equal(shouldShowFirstLibraryHint([]), true);
  assert.equal(shouldShowFirstLibraryHint([newLibrary(1)]), false);
  assert.equal(shouldShowFirstLibraryHint(undefined), false);
});

// --- Folder-first add: GET /api/library-inspect prefills the card. ---

test("a picked folder prefills the card from the detection endpoint", async () => {
  const calls = [];
  const requestJSON = async (url) => {
    calls.push(url);
    return {
      data: {
        path: "/media/Books",
        suggested: { name: "Library Books", id: "library-books", kind: "comics", storage: "filesystem" },
        evidence: ["12 EPUB files", "No Calibre database"],
        counts: { books: 0, comics: 12, audio: 0, video: 0, documents: 0, images: 0 },
      },
    };
  };
  const { library, evidence } = await inspectFolderLibrary("/media/Books", { requestJSON, number: 3 });

  assert.deepEqual(calls, ["/api/library-inspect?path=%2Fmedia%2FBooks"]);
  assert.equal(library.id, "new-library-3");
  assert.equal(library.name, "Library Books");
  assert.equal(library.kind, "comics");
  assert.equal(library.source.type, "filesystem");
  assert.equal(library.source.path, "/media/Books");
  assert.deepEqual(library.preferred_formats, ["cbz", "pdf", "epub"]);
  assert.equal(evidence, "12 EPUB files · No Calibre database");
  assert.equal("metadata_provider" in library, false, "the card never chooses a provider");
});

test("a Calibre suggestion becomes a Calibre library", async () => {
  const requestJSON = async () => ({
    data: {
      path: "/media/Calibre Library",
      suggested: { name: "Calibre Library", kind: "books", storage: "calibre" },
      evidence: ["Calibre database found", "2,104 books"],
    },
  });
  const { library, evidence } = await inspectFolderLibrary("/media/Calibre Library", { requestJSON });
  assert.equal(library.source.type, "calibre");
  assert.equal(library.source.path, "/media/Calibre Library");
  assert.equal(library.kind, "books");
  assert.equal(library.preferred_formats.join(","), "epub,pdf,mobi,azw3");
  assert.match(evidence, /Calibre database found/);
});

test("a suggestion never makes a document library use Calibre", () => {
  const library = libraryFromInspection("/notes", {
    suggested: { name: "Notes", kind: "documents", storage: "calibre" },
  });
  assert.equal(library.source.type, "filesystem");
});

test("an unknown or missing suggestion falls back to the folder itself", async () => {
  const empty = await inspectFolderLibrary("/srv/Library", { requestJSON: async () => ({ data: null }) });
  assert.equal(empty.library.name, "Library");
  assert.equal(empty.library.kind, "books");
  assert.equal(empty.library.source.type, "filesystem");
  assert.equal(empty.library.source.path, "/srv/Library");
  assert.equal(empty.evidence, "");

  const odd = libraryFromInspection("/srv/Things", { suggested: { kind: "spaceships" } });
  assert.equal(odd.kind, "books");
  assert.equal(odd.name, "Things");
});

test("a failed inspection still creates the card with the chosen path", async () => {
  const failing = async () => {
    throw new Error("offline");
  };
  const { library, evidence } = await inspectFolderLibrary("/media/my_books/", { requestJSON: failing, number: 2 });
  assert.equal(library.source.path, "/media/my_books/");
  assert.equal(library.name, "my_books");
  assert.equal(library.kind, "books");
  assert.equal(library.source.type, "filesystem");
  assert.deepEqual(library.preferred_formats, defaultPreferredFormats("books"));
  assert.equal(evidence, "");
  assert.equal("metadata_provider" in library, false);

  // No request helper at all (a card added without a picker) behaves the same.
  const offline = await inspectFolderLibrary("/media/Books");
  assert.equal(offline.library.source.path, "/media/Books");
  assert.equal(offline.library.name, "Books");
});

test("names from paths and evidence lines stay short and readable", () => {
  assert.equal(libraryNameFromPath("/media/Books/"), "Books");
  assert.equal(libraryNameFromPath("/media/Calibre Library"), "Calibre Library");
  assert.equal(libraryNameFromPath("/"), "New library");
  assert.equal(libraryNameFromPath(""), "New library");
  assert.equal(inspectionEvidence({ evidence: [" 12 EPUB files ", "", null, "No Calibre database"] }), "12 EPUB files · No Calibre database");
  assert.equal(inspectionEvidence({}), "");
  assert.equal(inspectionEvidence(null), "");
});

// --- Online metadata is derived, never a per-library field. ---

test("the metadata line follows the library kind and the saved tokens", () => {
  const none = {};
  const hardcoverOnly = { hardcover_token_configured: true, tmdb_token_configured: false };
  const both = { hardcover_token_configured: true, tmdb_token_configured: true };

  for (const kind of ["books", "comics", "manga", "mixed"]) {
    assert.equal(derivedMetadataProvider(kind, none), "Open Library");
    assert.equal(metadataLineText({ kind }, none), "Metadata: Open Library");
    assert.equal(metadataLineText({ kind }, hardcoverOnly), "Metadata: Hardcover");
    assert.equal(metadataLineText({ kind }, both), "Metadata: Hardcover");
  }
  for (const kind of ["movies", "tv", "anime"]) {
    assert.equal(metadataLineText({ kind }, none), "Metadata: Off");
    assert.equal(metadataLineText({ kind }, hardcoverOnly), "Metadata: Off");
    assert.equal(metadataLineText({ kind }, both), "Metadata: TMDB");
  }
  for (const kind of ["audiobooks", "podcasts", "music", "documents", "vault"])
    assert.equal(metadataLineText({ kind }, both), "Metadata: Off");
  assert.equal(metadataLineText(undefined, both), "Metadata: Off");
  // A saved name outranks the derivation only while it still fits the kind and
  // has its token — the same rule the server applies, so the line cannot claim a
  // provider the lookup is not using.
  assert.equal(
    metadataLineText({ kind: "books", metadata_provider: "openlibrary" }, both),
    "Metadata: Open Library",
  );
  assert.equal(
    metadataLineText({ kind: "books", metadata_provider: "off" }, both),
    "Metadata: Off",
  );
  assert.equal(
    metadataLineText({ kind: "audiobooks", metadata_provider: "hardcover" }, both),
    "Metadata: Off",
  );
  assert.equal(
    metadataLineText({ kind: "books", metadata_provider: "hardcover" }, none),
    "Metadata: Open Library",
  );
});

test("libraries never carry a metadata_provider choice of their own", () => {
  assert.equal("metadata_provider" in newLibrary(1), false);
  const picked = libraryFromInspection("/media/Books", {
    suggested: { name: "Books", kind: "books", storage: "filesystem" },
  });
  assert.equal("metadata_provider" in picked, false);
});
