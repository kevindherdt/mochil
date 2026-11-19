import assert from "node:assert/strict";
import test from "node:test";
import {
  metadataReviewHTML,
  mountMetadataReview,
} from "../static/metadata-review-template.js";

test("metadata review template retains inspection and managed-ingest contracts", () => {
  for (const id of [
    "metadata-intro",
    "metadata-library",
    "metadata-filter-wrap",
    "metadata-filter",
    "music-quality-filter-wrap",
    "music-quality-filter",
    "inspect-metadata",
    "preview-ingest",
    "ingest-preview-status",
    "ingest-preview-results",
    "ingest-preview-sentinel",
    "ingest-preview-more",
    "managed-ingest-workflow",
    "managed-ingest-kind",
    "managed-ingest-discover",
    "managed-ingest-discovery-status",
    "managed-ingest-discovery-results",
    "managed-ingest-discovery-more",
    "managed-ingest-batch-actions",
    "managed-ingest-batch-count",
    "managed-ingest-batch-preview",
    "managed-ingest-batch-clear",
    "managed-ingest-batch-review",
    "managed-ingest-batch-status",
    "managed-ingest-batch-entries",
    "managed-ingest-batch-plan",
    "managed-ingest-source",
    "managed-ingest-destination",
    "managed-ingest-plan",
    "managed-ingest-status",
    "managed-ingest-review",
    "managed-ingest-review-paths",
    "managed-ingest-confirmation",
    "managed-ingest-execute",
    "managed-ingest-discard",
    "metadata-status",
    "metadata-summary",
    "metadata-issues",
    "metadata-apply",
    "metadata-jobs",
  ])
    assert.match(metadataReviewHTML, new RegExp(`id="${id}"`));
  assert.match(metadataReviewHTML, /data-settings-panel="metadata">/);
  assert.doesNotMatch(metadataReviewHTML, /data-settings-panel="metadata" hidden/, "a section of Libraries, shown with it");
  assert.match(
    metadataReviewHTML,
    /Nothing changes until you review\s+and confirm it/,
  );
  assert.match(metadataReviewHTML, /Incoming files are never moved or deleted/);
  assert.ok((metadataReviewHTML.match(/role="status"/g) || []).length >= 5);
});

test("metadata review mounts once, inside the Libraries panel", () => {
  const insertions = [];
  let mounted = false;
  const settings = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === '[data-settings-panel="metadata"]'
        ? mounted
          ? {}
          : null
        : selector === '[data-settings-panel="libraries"]'
          ? settings
          : null;
    },
  };
  mountMetadataReview(document);
  mountMetadataReview(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: metadataReviewHTML },
  ]);
});
