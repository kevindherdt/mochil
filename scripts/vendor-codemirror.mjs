// Bundles CodeMirror 6 (MIT) into one ES module for the wiki's live-preview
// editor. Run after changing the pinned @codemirror versions in package.json;
// the output is committed and loaded only when someone edits a note.
import { build } from 'esbuild'
import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'

const out = 'internal/web/static/vendor/codemirror'
await mkdir(out, { recursive: true })
await build({
  stdin: {
    contents: `
      export { EditorState, Compartment, RangeSetBuilder } from '@codemirror/state'
      export { EditorView, Decoration, ViewPlugin, WidgetType, keymap, placeholder, drawSelection } from '@codemirror/view'
      export { history, historyKeymap, defaultKeymap, indentWithTab } from '@codemirror/commands'
      export { syntaxTree, HighlightStyle, syntaxHighlighting } from '@codemirror/language'
      export { autocompletion, completionKeymap, closeCompletion } from '@codemirror/autocomplete'
      export { markdown, markdownLanguage, markdownKeymap } from '@codemirror/lang-markdown'
    `,
    resolveDir: process.cwd(),
  },
  bundle: true, format: 'esm', target: 'es2022', minify: true, legalComments: 'none',
  outfile: `${out}/codemirror.js`,
})
await cp('node_modules/@codemirror/view/LICENSE', `${out}/LICENSE`)
const version = async name => JSON.parse(await readFile(`node_modules/@codemirror/${name}/package.json`, 'utf8')).version
await writeFile(`${out}/VERSION`, `state ${await version('state')}, view ${await version('view')}, commands ${await version('commands')}, language ${await version('language')}, lang-markdown ${await version('lang-markdown')}, autocomplete ${await version('autocomplete')}\n`)
