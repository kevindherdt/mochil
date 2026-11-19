import assert from "node:assert/strict";
import test from "node:test";
import {
  metadataReviewHTML,
  mountMetadataReview,
} from "../static/metadata-review-template.js";

test("metadata review template retains inspection contracts, not the retired managed-ingest workflow", () => {
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
  // The old manual-path-typing copy workflow was retired once the Import page
  // (0.1.18-0.1.20) made the same job a real folder picker; it must not come back.
  assert.doesNotMatch(metadataReviewHTML, /managed-ingest-workflow/);
  assert.doesNotMatch(metadataReviewHTML, />Import preview</, "renamed to avoid colliding with the Import feature");
  assert.match(metadataReviewHTML, />Metadata path preview</);
  assert.ok((metadataReviewHTML.match(/role="status"/g) || []).length >= 2);
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
