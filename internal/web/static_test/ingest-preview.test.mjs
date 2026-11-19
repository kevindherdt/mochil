import assert from "node:assert/strict";
import test from "node:test";
import { createIngestPreview, ingestPreviewItemHTML } from "../static/ingest-preview.js";

const escapeHTML = value => String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");

function fixture(requestJSON) {
  const elements = new Map();
  for (const id of ["metadata-library", "ingest-preview-results", "ingest-preview-status", "ingest-preview-more", "preview-ingest", "ingest-preview-sentinel"])
    elements.set(`#${id}`, { value: "books", hidden: false, disabled: false, textContent: "", innerHTML: "", children: [] });
  elements.get("#ingest-preview-results").insertAdjacentHTML = (_, markup) => {
    elements.get("#ingest-preview-results").innerHTML += markup;
    elements.get("#ingest-preview-results").children.push(...(markup.match(/<article/g) || []).map(() => ({})));
  };
  return { preview: createIngestPreview({ $: selector => elements.get(selector), escapeHTML, requestJSON, IntersectionObserverClass: null }), elements };
}

test("ingest preview markup escapes paths, titles, and proposed metadata", () => {
  const markup = ingestPreviewItemHTML({ path: "Series/<unsafe>.epub", changes: [{ field: "series_index", source: "folder", confidence: "high", proposed: "<2>" }] }, escapeHTML);
  assert.match(markup, /&lt;unsafe&gt;\.epub/);
  assert.match(markup, /series index/);
  assert.match(markup, /&lt;2&gt;/);
  assert.doesNotMatch(markup, /<unsafe>|<2>/);
});

test("ingest preview owns bounded continuation and reset state", async () => {
  const requests = [];
  const { preview, elements } = fixture(async path => {
    requests.push(path);
    return { data: requests.length === 1
      ? { inspected: 100, matched: 2, next_offset: 100, items: [{ title: "One", path: "One.epub", changes: [] }] }
      : { inspected: 130, matched: 2, next_offset: 0, items: [{ title: "Two", path: "Two.epub", changes: [] }] } };
  });
  await preview.load(true);
  await preview.load();
  assert.deepEqual(requests, [
    "/api/metadata/ingest-preview?library=books&offset=0&limit=100",
    "/api/metadata/ingest-preview?library=books&offset=100&limit=100",
  ]);
  assert.equal(elements.get("#ingest-preview-results").children.length, 2);
  assert.match(elements.get("#ingest-preview-status").textContent, /130 items checked/);
  assert.equal(elements.get("#preview-ingest").disabled, false);
  preview.reset();
  assert.equal(elements.get("#ingest-preview-results").innerHTML, "");
  assert.equal(elements.get("#ingest-preview-more").hidden, true);
});

test("ingest preview recovers controls after a failed request", async () => {
  const { preview, elements } = fixture(async () => { throw new Error("offline"); });
  await preview.load(true);
  assert.equal(elements.get("#preview-ingest").disabled, false);
  assert.equal(elements.get("#ingest-preview-more").hidden, true);
  assert.match(elements.get("#ingest-preview-status").textContent, /offline/);
});
