// The fields of a smart collection's rule, and how a rule is described on its card.
export const ruleFieldsHTML = `
  <fieldset id="collection-rule-fields" class="collection-rule" hidden>
    <legend>Fills itself with books that match</legend>
    <label>Search words<input id="rule-query" maxlength="200" autocomplete="off"></label>
    <label>Author<input id="rule-author" maxlength="200" autocomplete="off"></label>
    <label>Series<input id="rule-series" maxlength="200" autocomplete="off"></label>
    <label>Tag<input id="rule-tag" maxlength="200" autocomplete="off"></label>
    <label>Format<input id="rule-format" maxlength="20" placeholder="epub, pdf, cbz…" autocomplete="off"></label>
    <label>My reading<select id="rule-status"><option value="">Any</option><option value="unread">Not started</option><option value="reading">In progress</option><option value="finished">Finished</option></select></label>
    <label>Added in the last<select id="rule-added"><option value="0">Any time</option><option value="7">7 days</option><option value="30">30 days</option><option value="90">90 days</option><option value="365">Year</option></select></label>
    <label class="rule-check"><input id="rule-gaps" type="checkbox">Only series with missing volumes</label>
  </fieldset>`;

const FIELDS = { query: "#rule-query", author: "#rule-author", series: "#rule-series", tag: "#rule-tag", format: "#rule-format", status: "#rule-status" };

// fillRule puts a rule into the form; an empty rule clears it.
export function fillRule(root, rule = {}) {
  for (const [key, selector] of Object.entries(FIELDS)) root.querySelector(selector).value = rule[key] || "";
  root.querySelector("#rule-added").value = String(rule.added_days || 0);
  root.querySelector("#rule-gaps").checked = Boolean(rule.missing_volumes);
}

// readRule returns only what was filled in, as the server expects it.
export function readRule(root) {
  const rule = {};
  for (const [key, selector] of Object.entries(FIELDS)) {
    const value = root.querySelector(selector).value.trim();
    if (value) rule[key] = value;
  }
  const days = Number(root.querySelector("#rule-added").value);
  if (days > 0) rule.added_days = days;
  if (root.querySelector("#rule-gaps").checked) rule.missing_volumes = true;
  return rule;
}

const STATUS_WORDS = { unread: "Not started", reading: "In progress", finished: "Finished" };

// describeRule is the short line under a smart collection's name.
export function describeRule(rule = {}) {
  const parts = [];
  if (rule.status) parts.push(STATUS_WORDS[rule.status] || rule.status);
  if (rule.query) parts.push(`“${rule.query}”`);
  if (rule.author) parts.push(rule.author);
  if (rule.series) parts.push(rule.series);
  if (rule.tag) parts.push(`#${rule.tag}`);
  if (rule.format) parts.push(rule.format);
  if (rule.added_days) parts.push(`last ${rule.added_days} days`);
  if (rule.missing_volumes) parts.push("missing volumes");
  return parts.join(" · ");
}
