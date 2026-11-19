import test from "node:test";
import assert from "node:assert/strict";
import { createVaultTree } from "../static/vault-tree.js";
import { vaultPageEntries } from "../static/vault-pages.js";

test('vault sidebar hides attachments without removing them from the catalogue', () => {
  const entries = [
    {kind:'folder',path:'Assets'},
    {id:'page',path:'Page.md',format:'md',title:'Page'},
    {id:'long',path:'Long.markdown',format:'markdown',title:'Long'},
    {id:'image',path:'Assets/cover.png',format:'png',title:'Cover'},
    {id:'pdf',path:'Reference.pdf',format:'pdf',title:'Reference'},
  ];
  assert.deepEqual(vaultPageEntries(entries).map(entry=>entry.path),['Assets','Page.md','Long.markdown']);
  const html=createVaultTree({escapeHTML})(entries);
  assert.match(html,/data-document="page"/);
  assert.doesNotMatch(html,/data-document="(?:image|pdf)"/);
  assert.equal(entries.length,5);
});

const escapeHTML = (value) =>
  String(value).replace(
    /[&<>"']/g,
    (character) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        character
      ],
  );

test("vault tree preserves nested folders and root pages", () => {
  const render = createVaultTree({ escapeHTML });
  const html = render([
    { id: "root", title: "Home", path: "Home.md" },
    { id: "child", title: "Chapter", path: "Books/Novel/Chapter.md" },
  ]);
  assert.match(html, /data-document="root"/);
  assert.match(html, />Books</);
  assert.match(html, />Novel</);
  assert.doesNotMatch(html, /data-document="child"/, "a closed folder is drawn empty until it is opened");
  const box = { childElementCount: 0, innerHTML: "" };
  render.expand({ dataset: { vaultFolder: "Books/Novel" }, querySelector: () => box });
  assert.match(box.innerHTML, /data-document="child"/);
  box.childElementCount = 1;
  box.innerHTML = "kept";
  render.expand({ dataset: { vaultFolder: "Books/Novel" }, querySelector: () => box });
  assert.equal(box.innerHTML, "kept", "opening it again does not redraw");
});

test("folders holding the open page, and small top-level folders, start open", () => {
  const render = createVaultTree({ escapeHTML });
  const entries = [
    { id: "a", title: "A", path: "Top/Deep/A.md" },
    { id: "b", title: "B", path: "Top/Other/B.md" },
  ];
  const html = render(entries, "a");
  assert.match(html, /data-document="a"/);
  assert.doesNotMatch(html, /data-document="b"/);
});

test("vault tree escapes vault-controlled names", () => {
  const render = createVaultTree({ escapeHTML });
  const html = render([
    { id: "safe", title: "<script>", path: "Bad & Folder/<script>.md" },
  ]);
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /Bad &amp; Folder/);
});

test("vault tree follows Obsidian's folders-first natural name order", () => {
  const render = createVaultTree({ escapeHTML });
  const html = render([
    { id: "note10", title: "Note 10", path: "Note 10.md" },
    { id: "folder10", title: "Inside", path: "Folder 10/Inside.md" },
    { id: "note2", title: "Note 2", path: "Note 2.md" },
    { id: "folder2", title: "Inside", path: "Folder 2/Inside.md" },
  ]);
  assert.ok(html.indexOf(">Folder 2<") < html.indexOf(">Folder 10<"));
  assert.ok(html.indexOf(">Folder 10<") < html.indexOf(">Note 2<"));
  assert.ok(html.indexOf(">Note 2<") < html.indexOf(">Note 10<"));
});
