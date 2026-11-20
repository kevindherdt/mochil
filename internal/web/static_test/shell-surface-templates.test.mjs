import assert from "node:assert/strict";
import test from "node:test";
import {
  activityPanelHTML,
  mountActivityPanel,
} from "../static/activity-panel-template.js";
import {
  dashboardHTML,
  dashboardUndoHTML,
  mountDashboard,
} from "../static/dashboard-template.js";
import { readFileSync } from "node:fs";

const dashboardStyles = readFileSync(
  new URL("../static/dashboard.css", import.meta.url),
  "utf8",
);

function element(tagName) {
  return {
    tagName,
    setAttribute(name, value) {
      this[name] = value;
    },
  };
}

test("activity panel retains scan and status contracts", () => {
  for (const id of ["scan-all-libraries", "activity-message", "activity-list"])
    assert.match(activityPanelHTML, new RegExp(`id="${id}"`));
  assert.match(activityPanelHTML, /role="status"/);
});

test("activity panel mounts once after navigation", () => {
  const mounted = [],
    nav = {
      insertAdjacentElement(position, node) {
        mounted.push({ position, node });
      },
    };
  const document = {
    querySelector(selector) {
      if (selector === "nav") return nav;
      return (
        mounted.find((value) => `#${value.node.id}` === selector)?.node || null
      );
    },
    createElement: element,
  };
  mountActivityPanel(document);
  mountActivityPanel(document);
  assert.equal(mounted.length, 1);
  assert.equal(mounted[0].position, "afterend");
  assert.equal(mounted[0].node.hidden, true);
  assert.equal(mounted[0].node["aria-label"], "Library activity");
});

test("dashboard retains reading, watching, listening, and undo contracts", () => {
  // Home is what is next. A shelf of finished books belongs in the library's filters and behind the
  // Continue reading toggle, not as a second list on the first screen.
  assert.doesNotMatch(dashboardHTML, /id="finished-rail"/);
  assert.doesNotMatch(dashboardHTML, /id="finished-track"/);
  assert.match(dashboardHTML, /id="held-rail"/);
  assert.match(dashboardHTML, /class="dashboard-lanes"/);
  assert.match(dashboardHTML, /class="dashboard-media-lane"/);
  assert.doesNotMatch(dashboardHTML, /<h1>Home<\/h1>/);
  for (const id of [
    "video-reading-rail",
    "video-reading-track",
    "audio-reading-rail",
    "audio-reading-track",
    "dashboard-empty",
  ])
    assert.match(dashboardHTML, new RegExp(`id="${id}"`));
  assert.match(dashboardUndoHTML, /id="undo-message"/);
  assert.match(dashboardUndoHTML, /id="undo-action"/);
});

test("dashboard empty state represents every resumable media type", () => {
  assert.match(dashboardHTML, /Media you start will wait for you here\./);
  assert.doesNotMatch(dashboardHTML, /Books you start will wait for you here\./);
});

test("wide dashboard lanes wrap cards instead of clipping a horizontal rail", () => {
  assert.match(dashboardStyles, /@media\(min-width:1180px\)/);
  assert.match(
    dashboardStyles,
    /\.dashboard-lanes>\.dashboard-reading \.reading-track\{[^}]*grid-auto-flow:row[^}]*grid-template-columns:repeat\(2,minmax\(0,1fr\)\)[^}]*overflow-x:visible/s,
  );
});

test("dashboard and undo surface mount once in their correct containers", () => {
  const mounted = new Map(),
    main = {
      prepend(node) {
        mounted.set(`#${node.id}`, node);
      },
    },
    body = {
      append(node) {
        mounted.set(`#${node.id}`, node);
      },
    };
  const document = {
    querySelector(selector) {
      return selector === "main" ? main : mounted.get(selector) || null;
    },
    createElement: element,
    body,
  };
  mountDashboard(document);
  mountDashboard(document);
  assert.equal(mounted.size, 3);
  assert.equal(mounted.get("#app-problem").hidden, true, "the load-problem panel starts hidden");
  assert.equal(mounted.get("#dashboard-view").tagName, "section");
  assert.equal(mounted.get("#undo-toast").hidden, true);
  assert.equal(mounted.get("#undo-toast").role, "status");
});
