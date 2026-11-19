import assert from "node:assert/strict";
import test from "node:test";
import {
  collectionDialogHTML,
  mountCollectionDialog,
} from "../static/collection-dialog-template.js";
import {
  metadataApplyDialogHTML,
  mountMetadataApplyDialog,
} from "../static/metadata-apply-dialog-template.js";

function fakeDocument() {
  const appended = [];
  return {
    appended,
    querySelector(selector) {
      return appended.find((node) => `#${node.id}` === selector) || null;
    },
    createElement(tagName) {
      const attributes = {};
      return {
        tagName,
        attributes,
        setAttribute(name, value) {
          attributes[name] = value;
        },
      };
    },
    body: {
      append(node) {
        appended.push(node);
      },
    },
  };
}

test("collection dialog retains controller and accessibility contracts", () => {
  for (const id of [
    "collection-choices",
    "collection-name",
    "create-collection",
    "collection-dialog-status",
  ])
    assert.match(collectionDialogHTML, new RegExp(`id="${id}"`));
  assert.match(collectionDialogHTML, /aria-label="New collection name"/);
  assert.match(collectionDialogHTML, /role="status"/);
  assert.match(collectionDialogHTML, /<h2 id="collection-dialog-title">Save to collection<\/h2>/);
});

test("collection dialog mounts idempotently", () => {
  const document = fakeDocument();
  mountCollectionDialog(document);
  mountCollectionDialog(document);
  assert.equal(document.appended.length, 1);
  assert.equal(document.appended[0].tagName, "dialog");
  assert.equal(document.appended[0].className, "collection-dialog");
  // The dialog's name comes from the visible heading.
  assert.equal(document.appended[0].attributes["aria-labelledby"], "collection-dialog-title");
});

test("metadata apply dialog retains confirmation and safety contracts", () => {
  for (const id of [
    "metadata-confirmation-label",
    "metadata-confirmation",
    "confirm-metadata-apply",
    "metadata-apply-status",
  ])
    assert.match(metadataApplyDialogHTML, new RegExp(`id="${id}"`));
  assert.match(metadataApplyDialogHTML, /confirm-metadata-apply" disabled/);
  assert.match(metadataApplyDialogHTML, /entire job rolls back/);
});

test("metadata apply dialog mounts idempotently", () => {
  const document = fakeDocument();
  mountMetadataApplyDialog(document);
  mountMetadataApplyDialog(document);
  assert.equal(document.appended.length, 1);
  assert.equal(document.appended[0].tagName, "dialog");
  assert.equal(document.appended[0].className, "apply-dialog");
});
