import test from "node:test";
import assert from "node:assert/strict";
import { closeOpenMenus } from "../static/popup-controller.js";

test("popup dismissal keeps the menu containing the pointer", () => {
  const inside = { open: true, contains: (target) => target === "inside" };
  const outside = { open: true, contains: () => false };
  closeOpenMenus([inside, outside], "inside");
  assert.equal(inside.open, true);
  assert.equal(outside.open, false);
  closeOpenMenus([inside]);
  assert.equal(inside.open, false);
});
