import test from "node:test";
import assert from "node:assert/strict";
import {
  libraryFormHTML,
  mountLibraryFormTemplate,
} from "../static/library-form-template.js";

test("library form template retains every settings contract", () => {
  for (const field of [
    "name",
    "kind",
    "source.type",
    "source.path",
    "metadata_writes",
    "inference.folder_layout",
    "preferred_formats",
    "page_streaming",
  ]) {
    assert.match(
      libraryFormHTML,
      new RegExp(`data-field="${field.replace(".", "\\.")}"`),
    );
  }
  assert.match(libraryFormHTML, /class="scan-library"/);
  assert.match(libraryFormHTML, /class="remove"/);
  assert.match(libraryFormHTML, /Local folder/);
});

test("online metadata is reported, never chosen per library", () => {
  assert.doesNotMatch(libraryFormHTML, /data-field="metadata_provider"/);
  assert.doesNotMatch(libraryFormHTML, /Online metadata/);
  assert.match(libraryFormHTML, /data-metadata-line/);
});

test("the card keeps Name, Content, Storage and Folder path up front", () => {
  const body = libraryFormHTML.indexOf('class="library-form-body"');
  const more = libraryFormHTML.indexOf('class="library-advanced"');
  assert.ok(body >= 0 && more > body);
  const front = libraryFormHTML.slice(body, more);
  for (const field of ["name", "kind", "source.type", "source.path"])
    assert.match(front, new RegExp(`data-field="${field.replace(".", "\\.")}"`));
  for (const field of ["scan_every", "metadata_writes", "preferred_formats"])
    assert.doesNotMatch(front, new RegExp(`data-field="${field}"`));
});

test("More options holds Rescan, the metadata line and page streaming before Advanced", () => {
  const more = libraryFormHTML.indexOf('class="library-advanced"');
  const nested = libraryFormHTML.indexOf('class="library-advanced-more"');
  assert.ok(more > 0 && nested > more);
  const options = libraryFormHTML.slice(more, nested);
  assert.match(options, /<summary>More options<\/summary>/);
  assert.match(options, /data-field="scan_every"/);
  assert.match(options, /data-metadata-line/);
  assert.match(options, /data-field="page_streaming"/);
  // Data-safety switches stay out of the nested section.
  assert.match(options, /data-field="vault_writes"/);
  assert.match(options, /Allow metadata repairs/);
  assert.match(options, /Allow Calibre repairs/);
  assert.doesNotMatch(options, /data-field="preferred_formats"/);
  assert.doesNotMatch(options, /data-field="inference\.folder_layout"/);
});

test("Advanced is one level deeper and holds only the rare layout fields", () => {
  const nested = libraryFormHTML.indexOf('class="library-advanced-more"');
  assert.ok(nested > 0);
  const advanced = libraryFormHTML.slice(nested);
  assert.match(advanced, /<summary>Advanced<\/summary>/);
  assert.match(advanced, /data-field="preferred_formats"/);
  assert.match(advanced, /data-field="inference\.folder_layout"/);
  assert.doesNotMatch(advanced, /data-field="page_streaming"/);
  assert.doesNotMatch(advanced, /data-field="metadata_writes"/);
});

test("library ids are never asked of the user", () => {
  assert.doesNotMatch(libraryFormHTML, /data-field="id"/);
});

test("library form template mounts once", () => {
  const appended = [];
  const document = {
    getElementById: () => appended.find((node) => node.id === "library-form"),
    createElement: () => ({}),
    body: { append: (node) => appended.push(node) },
  };
  mountLibraryFormTemplate(document);
  mountLibraryFormTemplate(document);
  assert.equal(appended.length, 1);
  assert.equal(appended[0].id, "library-form");
  assert.equal(appended[0].innerHTML, libraryFormHTML);
});
