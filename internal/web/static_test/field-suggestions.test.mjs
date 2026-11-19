import assert from "node:assert/strict";
import test from "node:test";
import { lastSegment, startsWithFirst, withLastSegment } from "../static/field-suggestions.js";

test("names that begin with the typed letters come first, each group keeping its order", () => {
  const items = ["Alan B. Lloyd", "Brandon Sanderson", "Andy Weir", "Brian McClellan"].map((name) => ({ name, count: 1 }));
  assert.deepEqual(startsWithFirst(items, "an").map((item) => item.name), ["Andy Weir", "Alan B. Lloyd", "Brandon Sanderson", "Brian McClellan"]);
});

test("the match is not case sensitive", () => {
  const items = [{ name: "Wilbur Smith", count: 7 }, { name: "Will Wight", count: 20 }];
  assert.equal(startsWithFirst(items, "WIL")[0].name, "Wilbur Smith");
});

test("a cell with several values suggests for the last one and replaces only that one", () => {
  assert.equal(lastSegment("Neil Gaiman & Terry Pr", "&"), "Terry Pr");
  assert.equal(withLastSegment("Neil Gaiman & Terry Pr", "&", "Terry Pratchett"), "Neil Gaiman & Terry Pratchett");
  assert.equal(lastSegment("sci-fi, ep", ","), "ep");
  assert.equal(withLastSegment("sci-fi, ep", ",", "epic"), "sci-fi, epic");
  assert.equal(withLastSegment("An", "", "Andy Weir"), "Andy Weir");
  assert.equal(withLastSegment("Gaiman", "&", "Neil Gaiman"), "Neil Gaiman");
});
