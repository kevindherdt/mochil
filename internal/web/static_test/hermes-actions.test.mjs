import test from "node:test";
import assert from "node:assert/strict";
import { plainText, bookLine, wrapContext, splitContext, gather, ACTIONS } from "../static/hermes-actions.js";

test("plainText removes markup and entities and cuts at a word", () => {
  assert.equal(plainText("<p>Sacred &amp; safe&nbsp;valley.</p> <p>Next</p>"), "Sacred & safe valley. Next");
  const cut = plainText("word ".repeat(300), 40);
  assert.ok(cut.endsWith("…") && cut.length <= 41);
  assert.equal(plainText(undefined), "");
});

test("a book is one line with series and progress", () => {
  assert.equal(bookLine({ title: "Reaper", authors: ["Will Wight"], series: "Cradle", series_index: 10, reading_progress: { fraction: 0.4 } }), "- Reaper by Will Wight (Cradle #10), 40% read");
  assert.equal(bookLine({ title: "1984", authors: [], reading_progress: { finished: true, fraction: 1 } }), "- 1984, finished");
  assert.equal(bookLine({ title: "Plain" }), "- Plain");
});

test("context is folded into a message and can be split off again", () => {
  const message = wrapContext("What's new?", "- A\n- B");
  assert.deepEqual(splitContext(message), { prompt: "What's new?", context: "- A\n- B" });
  assert.equal(wrapContext("Hi", ""), "Hi");
  assert.deepEqual(splitContext("Just a message"), { prompt: "Just a message", context: "" });
});

test("each action gathers only its own context from the catalogue", async () => {
  const calls = [];
  const api = async (url) => { calls.push(url); return url.startsWith("/api/reading?finished") ? [{ title: "Done" }] : url.startsWith("/api/reading") ? [{ title: "Now", reading_progress: { fraction: 0.5 } }] : [{ title: "New" }]; };
  const byId = (id) => ACTIONS.find((a) => a.id === id);
  assert.equal(await gather(byId("recent"), api), "Recently added books:\n- New");
  const next = await gather(byId("next"), api);
  assert.ok(next.includes("Reading now:\n- Now, 50% read") && next.includes("Recently finished:\n- Done") && next.includes("Recently added:\n- New"));
  const book = { title: "Reaper", authors: ["Will Wight"], library_name: "Books", description: "<p>Lindon delves.</p>", language: "en" };
  assert.equal(await gather(byId("book"), api, book), "- Reaper by Will Wight\nLibrary: Books, language: en\nDescription: Lindon delves.");
  assert.equal(await gather(byId("polish"), api), "");
  assert.ok(calls.every((url) => url.startsWith("/api/books") || url.startsWith("/api/reading")), "only the catalogue is read");
});

test("a failing part of the context does not stop the question", async () => {
  const api = async (url) => { if (url.startsWith("/api/reading")) throw new Error("down"); return [{ title: "New" }]; };
  const text = await gather(ACTIONS.find((a) => a.id === "next"), api);
  assert.equal(text, "Recently added:\n- New");
});
