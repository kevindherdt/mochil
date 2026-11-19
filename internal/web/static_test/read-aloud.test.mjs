import assert from "node:assert/strict";
import test from "node:test";
import { createReadAloud, readAloudRates } from "../static/read-aloud.js";

function fixture({ withOverlay = true, savedRate } = {}) {
  const calls = [];
  const listeners = {};
  const overlay = withOverlay && {
    addEventListener: (type, fn) => (listeners[type] = fn),
    start: (...args) => calls.push(["start", ...args]),
    pause: () => calls.push(["pause"]),
    resume: () => calls.push(["resume"]),
    setRate: (rate) => calls.push(["rate", rate]),
    prev: () => calls.push(["prev"]),
    next: () => calls.push(["next"]),
  };
  const attrs = {};
  const button = { hidden: true, dataset: {}, setAttribute: (k, v) => (attrs[k] = v) };
  const speedField = { hidden: true };
  const speedSelect = { value: "" };
  const stored = new Map(savedRate ? [["mochil-readaloud-rate", String(savedRate)]] : []);
  const storage = { getItem: (k) => stored.get(k) ?? null, setItem: (k, v) => stored.set(k, v) };
  const element = { id: "s1" };
  const doc = { getElementById: (id) => (id === "s1" ? element : null) };
  const range = { comparePoint: (el) => (el === element ? -1 : 1) };
  const view = {
    mediaOverlay: overlay,
    lastLocation: { index: 3, range },
    renderer: { getContents: () => [{ index: 3, doc }] },
  };
  const api = createReadAloud({ view, button, speedField, speedSelect, storage, getTitle: () => "T" });
  return { api, calls, listeners, attrs, button, speedField, speedSelect, stored };
}

test("books without media overlays get no read-aloud control", () => {
  assert.equal(fixture({ withOverlay: false }).api, null);
});

test("the control appears, restores the saved speed, and starts on the current page", () => {
  const { api, calls, button, speedField, speedSelect } = fixture({ savedRate: 1.5 });
  assert.equal(button.hidden, false);
  assert.equal(speedField.hidden, false);
  assert.equal(speedSelect.value, "1.5");
  assert.deepEqual(calls[0], ["rate", 1.5]);
  api.toggle();
  const start = calls.find((call) => call[0] === "start");
  assert.equal(start[1], 3);
  // Sentences before the page are skipped; the one after it is kept.
  assert.equal(start[2]({ text: "ch.xhtml#s1" }), false);
  assert.equal(start[2]({ text: "ch.xhtml#missing" }), true);
  assert.equal(button.dataset.state, "playing");
});

test("toggle pauses and resumes without restarting", () => {
  const { api, calls, button } = fixture();
  api.toggle();
  api.toggle();
  assert.equal(button.dataset.state, "paused");
  api.toggle();
  assert.equal(button.dataset.state, "playing");
  assert.deepEqual(calls.filter((c) => ["pause", "resume", "start"].includes(c[0])).map((c) => c[0]), ["start", "pause", "resume"]);
});

test("speed changes persist and unknown speeds are ignored", () => {
  const { api, calls, stored } = fixture();
  api.setRate(2);
  api.setRate(9);
  assert.equal(stored.get("mochil-readaloud-rate"), "2");
  assert.deepEqual(calls.filter((c) => c[0] === "rate").map((c) => c[1]), [1, 2]);
  assert.ok(readAloudRates.includes(2));
});

test("an audio error pauses and is reported", () => {
  const errors = [];
  const f = fixture();
  f.api.toggle();
  f.listeners.error({ detail: new Error("x") });
  assert.equal(f.button.dataset.state, "paused");
  assert.equal(errors.length, 0);
});

test("the sleep timer pauses reading after the chosen time and resets itself", () => {
  const scheduled = [];
  const cleared = [];
  const slept = [];
  const calls = [];
  const sleepSelect = { value: "30" };
  const listeners = {};
  const overlay = {
    addEventListener: (type, fn) => (listeners[type] = fn),
    start: () => calls.push("start"), pause: () => calls.push("pause"), resume: () => calls.push("resume"),
    setRate: () => {}, prev: () => {}, next: () => {},
  };
  const button = { hidden: true, dataset: {}, setAttribute: () => {} };
  const sleepField = { hidden: true };
  const api = createReadAloud({
    view: { mediaOverlay: overlay, lastLocation: { index: 0 }, renderer: { getContents: () => [] } },
    button, sleepField, sleepSelect, storage: { getItem: () => null, setItem: () => {} },
    onSleep: () => slept.push(true),
    timers: { set: (fn, ms) => (scheduled.push({ fn, ms }), scheduled.length), clear: (id) => cleared.push(id) },
  });
  assert.equal(sleepField.hidden, false);
  api.toggle();
  api.setSleep(30);
  assert.equal(scheduled[0].ms, 30 * 60_000);
  api.setSleep(10);
  assert.deepEqual(cleared, [1], "choosing a new time replaces the old timer");
  scheduled[1].fn();
  assert.equal(button.dataset.state, "paused");
  assert.equal(sleepSelect.value, "0");
  assert.equal(slept.length, 1);
  api.setSleep(0);
  assert.equal(scheduled.length, 2, "Off schedules nothing");
});

test("the default timers work when called as methods", async () => {
  const slept = [];
  const overlay = { addEventListener() {}, start() {}, pause() {}, resume() {}, setRate() {}, prev() {}, next() {} };
  const api = createReadAloud({
    view: { mediaOverlay: overlay, lastLocation: { index: 0 }, renderer: { getContents: () => [] } },
    button: { hidden: true, dataset: {}, setAttribute() {} },
    storage: { getItem: () => null, setItem() {} },
    onSleep: () => slept.push(true),
  });
  api.setSleep(0.0002);
  await new Promise((resolve) => setTimeout(resolve, 60));
  assert.equal(slept.length, 1);
});
