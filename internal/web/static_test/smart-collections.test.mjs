import assert from "node:assert/strict";
import test from "node:test";
import { createCollections } from "../static/collections.js";

function fixture({ library = "books", existing = [] } = {}) {
  const nodes = {};
  const element = () => ({ setAttribute() {}, addEventListener() {}, append() {}, replaceChildren() {}, children: [] });
  const node = (selector) => (nodes[selector] ||= { ...element(), parentElement: element(), textContent: "", innerHTML: "", value: "", hidden: false, showModal() { this.opened = true; } });
  node("#collection-dialog").querySelector = (selector) => node(selector);
  const heading = { textContent: "" };
  const document = {
    querySelector: (selector) => (selector === "#collection-dialog h2" ? heading : selector === "#collection-dialog" ? node(selector) : null),
    createElement: element,
    body: { append() {} },
    dispatchEvent() {},
  };
  const requests = [];
  const requestJSON = async (url, options = {}) => {
    requests.push({ url, options });
    if (options.method === "POST") return { data: { id: "new" } };
    return { data: existing };
  };
  const api = createCollections({
    document, $: node, $$: () => [], escapeHTML: String, requestJSON, bookMarkup() {}, installCoverFallbacks() {},
    getSelectedLibrary: () => library, getItemLayout: () => "covers", isActive: () => false, abortBrowse() {},
  });
  api.bind();
  return { api, node, heading, requests };
}

test("saving a search keeps the words as a rule, in the library that is open", async () => {
  const { api, node, heading, requests } = fixture();
  await api.saveSearch("frank herbert");
  assert.equal(node("#collection-dialog").opened, true);
  assert.equal(heading.textContent, "Smart collection");
  assert.equal(node("#collection-name").value, "frank herbert");
  await node("#create-collection").onclick();
  const created = requests.find((r) => r.options.method === "POST");
  assert.deepEqual(JSON.parse(created.options.body), { name: "frank herbert", library_id: "books", rule: { query: "frank herbert" } });
});

test("a smart collection from the form sends every field that was filled in", async () => {
  const { api, node, requests } = fixture();
  await api.saveSearch("dune");
  node("#rule-status").value = "unread";
  node("#rule-added").value = "30";
  node("#rule-tag").value = "classic";
  await node("#create-collection").onclick();
  const created = requests.find((r) => r.options.method === "POST");
  assert.deepEqual(JSON.parse(created.options.body).rule, { query: "dune", tag: "classic", status: "unread", added_days: 30 });
});

test("an empty search or no library saves nothing", async () => {
  const empty = fixture();
  await empty.api.saveSearch("");
  assert.equal(empty.node("#collection-dialog").opened, undefined);
  const none = fixture({ library: "" });
  await none.api.saveSearch("dune");
  assert.equal(none.node("#collection-dialog").opened, undefined);
});

test("smart collections are not offered when saving a book", async () => {
  const { api, node } = fixture({ existing: [
    { id: "a", name: "Picked", count: 1, members: [] },
    { id: "b", name: "Smart one", count: 3, members: [], rule: { query: "dune" } },
  ] });
  await api.openFor([{ type: "book", library_id: "books", book_id: "x" }]);
  const choices = node("#collection-choices").innerHTML;
  assert.match(choices, /Picked/);
  assert.doesNotMatch(choices, /Smart one/);
});

test("the save-search button shows only for a search that found something in a library", () => {
  const { api, node } = fixture();
  const button = node("#save-search");
  node("#search").value = "";
  api.syncSaveSearch(true);
  assert.equal(button.hidden, true, "no search words");
  node("#search").value = "gaiman";
  api.syncSaveSearch(false);
  assert.equal(button.hidden, true, "nothing found");
  api.syncSaveSearch(true);
  assert.equal(button.hidden, false);
  assert.equal(typeof button.onclick, "function");
  const none = fixture({ library: "" });
  none.node("#search").value = "gaiman";
  none.api.syncSaveSearch(true);
  assert.equal(none.node("#save-search").hidden, true, "Home has no single library to save into");
});
