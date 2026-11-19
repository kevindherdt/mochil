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

The video player bundles hls.js (Apache-2.0); the license text is in
`internal/web/static/vendor/hls.js/LICENSE`.

The optional avatar module bundles PixiJS 6.5.10 (MIT) and the Cubism 4 build of
`pixi-live2d-display` (MIT, Copyright (c) 2020 Guan), both in
`internal/web/static/vendor/avatar/`, with their license texts beside them.

Mochil does **not** include the Live2D Cubism Core runtime. That library is
distributed by Live2D Inc. under its own proprietary license, so it is left out
of this repository on purpose. The avatar module expects an administrator to put
it in the avatar folder as `_runtime/live2dcubismcore.min.js`, where the server
serves it to that installation only; without it the module reports that the core
is missing and the rest of the app is unaffected. If you enable avatars, Live2D's
terms are between you and Live2D Inc.
