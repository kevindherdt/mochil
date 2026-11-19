import test from "node:test";
import assert from "node:assert/strict";
import { compactRelativeTime, relativePublishedDate } from "../static/date-format.js";
import { readFile } from "node:fs/promises";

const now = Date.parse("2026-08-27T12:00:00Z");

test("missing, zero, and malformed podcast dates remain undisplayed", () => {
  assert.equal(relativePublishedDate("", now), "");
  assert.equal(relativePublishedDate("0001-01-01T00:00:00Z", now), "");
  assert.equal(relativePublishedDate("not-a-date", now), "");
});

test("credible publication dates retain useful relative labels", () => {
  assert.equal(relativePublishedDate("2026-08-27T08:00:00Z", now), "today");
  assert.match(relativePublishedDate("2026-08-20T12:00:00Z", now), /7 days/);
  assert.match(relativePublishedDate("2024-08-27T12:00:00Z", now), /2 years/);
});

test("implausibly distant future dates are hidden", () => {
  assert.equal(relativePublishedDate("2030-01-01T00:00:00Z", now), "");
});

test("the compact dashboard style stays short across every bucket", () => {
  assert.equal(compactRelativeTime(now - 10_000, now), "Just now");
  assert.equal(compactRelativeTime(now - 5 * 60_000, now), "5m ago");
  assert.equal(compactRelativeTime(now - 3 * 3_600_000, now), "3h ago");
  assert.equal(compactRelativeTime(now - 2 * 86_400_000, now), "2d ago");
});

test("the compact dashboard style never goes negative for a clock-skewed future timestamp", () => {
  assert.equal(compactRelativeTime(now + 60_000, now), "Just now");
});

test("the dashboard reuses the shared relative-time module instead of its own copy", async () => {
  const source = await readFile(new URL("../static/reading-dashboard.js", import.meta.url), "utf8");
  assert.doesNotMatch(source, /function relativeTime\(/);
  assert.match(source, /compactRelativeTime/);
  assert.match(source, /from ['"]\.\/date-format\.js/);
});
