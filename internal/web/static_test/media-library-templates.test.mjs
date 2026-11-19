import assert from "node:assert/strict";
import test from "node:test";
import {
  mountVideoLibrary,
  videoLibraryHTML,
} from "../static/video-library-template.js";
import {
  documentLibraryHTML,
  mountDocumentLibrary,
} from "../static/document-library-template.js";

function insertionDocument(existingSelector) {
  const insertions = [];
  let mounted = false;
  const anchor = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  return {
    insertions,
    document: {
      querySelector(selector) {
        return selector === existingSelector ? (mounted ? {} : null) : anchor;
      },
    },
  };
}

test("video library template retains list and paging contracts", () => {
  for (const id of [
    "video-index",
    "video-top-spacer",
    "video-start-sentinel",
    "video-load-previous",
    "videos",
    "video-bottom-spacer",
    "video-sentinel",
    "video-load-more",
  ])
    assert.match(videoLibraryHTML, new RegExp(`id="${id}"`));
  assert.match(videoLibraryHTML, /video-index" hidden/);
  assert.match(videoLibraryHTML, /video-start-sentinel" aria-hidden="true"/);
  assert.match(videoLibraryHTML, /video-sentinel" aria-hidden="true"/);
});

test("video library mounts once before the shared collection anchor", () => {
  const { document, insertions } = insertionDocument("#video-index");
  mountVideoLibrary(document);
  mountVideoLibrary(document);
  assert.deepEqual(insertions, [
    { position: "beforebegin", markup: videoLibraryHTML },
  ]);
});

test("document library template retains sorting and paging contracts", () => {
  for (const id of [
    "document-index",
    "document-toolbar",
    "document-sort",
    "document-top-spacer",
    "document-start-sentinel",
    "document-load-previous",
    "documents",
    "document-bottom-spacer",
    "document-sentinel",
    "document-load-more",
  ])
    assert.match(documentLibraryHTML, new RegExp(`id="${id}"`));
  assert.match(documentLibraryHTML, /aria-label="Sort documents"/);
  assert.match(documentLibraryHTML, /document-index" hidden/);
  assert.match(documentLibraryHTML, /document-start-sentinel" aria-hidden="true"/);
});

test("document library mounts once before collections", () => {
  const { document, insertions } = insertionDocument("#document-index");
  mountDocumentLibrary(document);
  mountDocumentLibrary(document);
  assert.deepEqual(insertions, [
    { position: "beforebegin", markup: documentLibraryHTML },
  ]);
});
