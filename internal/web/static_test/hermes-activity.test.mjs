import test from "node:test";
import assert from "node:assert/strict";
import { createStreamReader, toolWords } from "../static/hermes-activity.js";

const lines = [
  'data: {"choices":[{"delta":{"role":"assistant"}}]}', "",
  "event: hermes.tool.progress",
  'data: {"tool":"web_search","label":"weather in Brussels","toolCallId":"t1","status":"running"}', "",
  "event: hermes.tool.progress",
  'data: {"tool":"web_search","toolCallId":"t1","status":"completed"}', "",
  'data: {"choices":[{"delta":{"content":"It is "}}]}', "",
  'data: {"choices":[{"delta":{"content":"7 degrees."}}]}', "",
  "data: [DONE]",
];

test("text and tool steps are told apart, and a step keeps its detail until it is done", () => {
  const reader = createStreamReader();
  const results = lines.map((line) => reader.feed(line)).filter(Boolean);
  assert.deepEqual(results.filter((r) => r.text).map((r) => r.text).join(""), "It is 7 degrees.");
  const stepEvents = results.filter((r) => r.steps);
  assert.equal(stepEvents.length, 2);
  assert.deepEqual(stepEvents[0].steps, [{ id: "t1", tool: "web_search", detail: "weather in Brussels", status: "running" }]);
  assert.deepEqual(stepEvents[1].steps, [{ id: "t1", tool: "web_search", detail: "weather in Brussels", status: "done" }]);
});

test("a tool event never leaks into the answer, and bad lines are ignored", () => {
  const reader = createStreamReader();
  reader.feed("event: hermes.tool.progress");
  assert.equal(reader.feed('data: {"tool":"x","choices":[{"delta":{"content":"LEAK"}}]}').text, undefined);
  reader.feed(""); // a blank line ends the event
  assert.equal(reader.feed("data: not json"), null);
  assert.equal(reader.feed(": keep-alive"), null);
  assert.equal(reader.feed('data: {"choices":[{"delta":{}}]}'), null);
});

test("failed steps are marked and tools get readable names", () => {
  const reader = createStreamReader();
  reader.feed("event: hermes.tool.progress");
  const { steps } = reader.feed('data: {"tool":"web_extract","toolCallId":"a","status":"failed"}');
  assert.equal(steps[0].status, "failed");
  assert.equal(toolWords("web_search"), "Searching the web");
  assert.equal(toolWords("terminal_run"), "Terminal run");
  assert.equal(toolWords("ha_call_service"), "Controlling the home");
  assert.equal(toolWords("ha_get_state"), "Checking a device");
});

test("a waiting signal shows as a step with the seconds, and goes away when the first words arrive", () => {
  const reader = createStreamReader();
  const wait = ["event: hermes.waiting", 'data: {"seconds":10}', ""].map((l) => reader.feed(l)).filter(Boolean);
  assert.equal(wait.length, 1);
  assert.deepEqual(wait[0].steps.map((s) => [s.tool, s.detail, s.status]), [["waiting", "10 s", "running"]]);
  const later = ["event: hermes.waiting", 'data: {"seconds":20}', ""].map((l) => reader.feed(l)).filter(Boolean);
  assert.equal(later[0].steps.length, 1);
  assert.equal(later[0].steps[0].detail, "20 s");
  const first = reader.feed('data: {"choices":[{"delta":{"content":"Hi"}}]}');
  assert.equal(first.text, "Hi");
  assert.deepEqual(first.steps, []);
  assert.deepEqual(reader.steps(), []);
  assert.equal(toolWords("waiting"), "Waiting");
});

test("a failure reported by the relay is handed to the page, and SSE comments are ignored", () => {
  const reader = createStreamReader();
  assert.equal(reader.feed(": ping"), null);
  const error = ["event: hermes.error", 'data: {"error":"Hermes is not reachable"}', ""].map((l) => reader.feed(l)).filter(Boolean);
  assert.deepEqual(error, [{ error: "Hermes is not reachable" }]);
});

test("the cards she offers to play come through as their own piece, and a bad one is ignored", () => {
  const reader = createStreamReader();
  const piece = ["event: hermes.cards", 'data: {"cards":[{"id":"1","title":"Lilac","stream_url":"/audio/music/1/a.mp3"}]}', ""].map((l) => reader.feed(l)).filter(Boolean);
  assert.equal(piece.length, 1);
  assert.equal(piece[0].cards[0].title, "Lilac");
  const odd = ["event: hermes.cards", 'data: {"cards":"nope"}', ""].map((l) => reader.feed(l)).filter(Boolean);
  assert.deepEqual(odd[0].cards, []);
});
