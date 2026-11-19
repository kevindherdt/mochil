import test from "node:test";
import assert from "node:assert/strict";
import { createHermesSettings, hermesFeatures } from "../static/hermes-settings.js";

// Enough of the Hermes panel for the settings writer: the fields it reads and the
// ability switches it counts. Nothing here talks to the server.
function fakeHermes() {
  const fields = new Map();
  const field = (selector) => {
    if (!fields.has(selector))
      fields.set(selector, { value: "", placeholder: "", textContent: "", hidden: false, addEventListener() {} });
    return fields.get(selector);
  };
  const switches = new Map(hermesFeatures.map((feature) => [feature.id, true]));
  const document = {
    querySelector(selector) {
      if (selector === "#settings-view" || selector === '[data-settings-tab="modules"]') return { insertAdjacentHTML() {} };
      return null;
    },
    querySelectorAll(selector) {
      if (selector !== "#hermes-features [data-feature]") return [];
      return [...switches].map(([id, checked]) => ({
        dataset: { feature: id },
        getAttribute: () => String(checked),
        setAttribute: (_, value) => switches.set(id, value === "true"),
      }));
    },
  };
  return { document, $: field, switches };
}

test("turning an editor tool off is saved even with no Hermes connection configured", () => {
  const { document, $, switches } = fakeHermes();
  const hermes = createHermesSettings({ document, $ });
  hermes.load({ modules: { hermes: true } }, [{ id: "hermes" }]);
  switches.set("isbn", false);
  const target = {};
  hermes.write(target);
  assert.deepEqual(target.hermes, { features: { isbn: false } });
});

test("turning it back on drops the stored exception again", () => {
  const { document, $, switches } = fakeHermes();
  const hermes = createHermesSettings({ document, $ });
  hermes.load({ modules: { hermes: true }, hermes: { features: { isbn: false } } }, [{ id: "hermes" }]);
  assert.equal(switches.get("isbn"), false, "load reflects the saved off switch");
  switches.set("isbn", true);
  const target = {};
  hermes.write(target);
  assert.equal(target.hermes, undefined, "nothing configured, nothing to persist");
});
