# Third-party notices

The browser reader vendors Foliate JS at commit
`78914aef4466eb960965702401634c2cb348e9b1` (2026-05-01). Foliate JS is MIT
licensed; its license is retained at
`internal/web/static/vendor/foliate-js/LICENSE`.

Foliate JS includes these runtime dependencies:

- zip.js 2.7.52 — BSD-3-Clause
- fflate 0.8.2 — MIT
- PDF.js 5.5.207 — Apache-2.0

The optional Readium preview reader bundles the Readium ts-toolkit
(`@readium/navigator` 2.14.0 and `@readium/shared` 2.8.0, with their
dependencies) under the BSD-3-Clause license. The license text is in
`internal/web/static/vendor/readium/LICENSE`; rebuild the bundle with
`node scripts/vendor-readium.mjs`.

The wiki note editor bundles CodeMirror 6 (`@codemirror/state`, `view`,
`commands`, `language`, `lang-markdown`, `autocomplete` and their Lezer dependencies) under the
MIT license. The license text is in
`internal/web/static/vendor/codemirror/LICENSE`; rebuild the bundle with
`node scripts/vendor-codemirror.mjs`.

The interface embeds Newsreader Variable 5.2.8 and Source Sans 3 Variable
5.2.8, both under the SIL Open Font License 1.1. Complete license texts are in
`internal/web/static/vendor/licenses/` and are embedded in the distributed
server binary with the corresponding assets.
