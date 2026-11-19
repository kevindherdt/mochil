import assert from "node:assert/strict";
import test from "node:test";
import {
  accountsSettingsHTML,
  mountAccountsSettings,
} from "../static/accounts-settings-template.js";

test("account settings template retains every controller contract", () => {
  for (const id of [
    "accounts-count",
    "add-account",
    "account-recovery",
    "account-recovery-form",
    "account-recovery-password",
    "confirm-account-migration",
    "account-create",
    "account-username",
    "account-role",
    "account-password",
    "create-account",
    "cancel-account-create",
    "accounts-status",
    "accounts-list",
  ])
    assert.match(accountsSettingsHTML, new RegExp(`id="${id}"`));
  assert.match(accountsSettingsHTML, /data-settings-panel="accounts" hidden/);
  assert.match(accountsSettingsHTML, /Public registration is disabled/);
});

test("account settings mount once inside settings", () => {
  const insertions = [];
  let mounted = false;
  const settings = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === '[data-settings-panel="accounts"]'
        ? mounted
          ? {}
          : null
        : settings;
    },
  };
  mountAccountsSettings(document);
  mountAccountsSettings(document);
  assert.equal(insertions.length, 1);
  assert.equal(insertions[0].position, "beforeend");
  assert.equal(insertions[0].markup, accountsSettingsHTML);
});
