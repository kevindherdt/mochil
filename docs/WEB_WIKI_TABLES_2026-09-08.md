# Web wiki pipe tables

Pending after 0.16.78; not deployed.

Wiki Markdown opts into a small dedicated pipe-table parser. It recognizes header/divider pairs, optional outside pipes, escaped pipes, and alignment markers; rendered cells use the existing escaped inline renderer, including wiki links. Header cells have column scope. A focusable, labeled horizontal scroll region contains wide tables without widening the page, styled with incumbent text/line/accent tokens.

Non-vault Markdown keeps its prior behavior. Fenced examples remain literal. Unsupported header syntax and rows above 64 columns fall back to ordinary Markdown; mismatched row widths terminate the table and remain visible as subsequent content. This is not a complete GFM parser; multiline cells and every pipe/code escape combination are not claimed. Row count remains governed by existing input size limits, not virtualization.

All 525 frontend tests and module checks passed. Chromium fixtures at 1440px and 390px passed header semantics, keyboard focus and no page overflow with twelve wide columns. These are focused interaction/layout checks, not a whole-app visual acceptance pass. Impeccable guided contained overflow, native table semantics and retained styling. No source notes were rewritten.

## Repeatable browser gate

`npm run test:browser:wiki` now runs the checked-in `scripts/browser-wiki-render-smoke.mjs` fixture at both widths using actual renderer modules and vault CSS. It checks properties disclosure with keyboard input, long-value page overflow, table header semantics and focus, and Unicode heading scroll/focus, while rejecting browser page errors. Both widths passed. Fixture palette/container styles are synthetic; this does not replace live-shell or physical-device acceptance.

## Escaped separator regression

The row scanner now counts consecutive backslashes before pipes. Odd runs escape the pipe; even runs retain a column boundary, matching the optional trailing-boundary handling. Regression tests cover run lengths zero through twelve and escaped pipes inside inline code. All 527 frontend tests, both browser widths and module checks passed. Not deployed.
