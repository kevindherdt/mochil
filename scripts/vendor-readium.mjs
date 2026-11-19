// Bundles the Readium ts-toolkit navigator (BSD-3-Clause) into one ES module
// that the optional Readium reader imports. Run after changing the pinned
// @readium versions in package.json; the output is committed.
import { build } from 'esbuild'
import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'

const out = 'internal/web/static/vendor/readium'
await mkdir(out, { recursive: true })

// Mochil patch: with "Reduce animations" on (the reader page's <html data-motion="reduced">), a
// swipe lands on the next page at once instead of sliding. The column snapper runs inside the book's
// own frame, so it reads the setting from the page that holds the frame (same origin).
const snapper = '/@readium/navigator-html-injectables/dist/modules/snapper/ColumnSnapper.js'
const stillSwipes = {
  name: 'bobo-still-swipes',
  setup(build) {
    build.onLoad({ filter: /ColumnSnapper\.js$/ }, async ({ path }) => {
      if (!path.endsWith(snapper)) return undefined
      const source = await readFile(path, 'utf8')
      const animated = 't&&a!==c){this.snappingCancelled=!1'
      if (!source.includes(animated)) throw new Error('ColumnSnapper changed: the reduce-animations patch no longer applies')
      const contents = source.replace(animated, 't&&a!==c&&this.wnd.parent?.document?.documentElement?.dataset?.motion!=="reduced"){this.snappingCancelled=!1')
      return { contents, loader: 'js' }
    })
  },
}
await build({
  stdin: {
    contents: `
      export { EpubNavigator } from '@readium/navigator'
      export { Manifest, Publication, HttpFetcher, Link, Locator } from '@readium/shared'
    `,
    resolveDir: process.cwd(),
  },
  bundle: true, format: 'esm', target: 'es2022', minify: true, legalComments: 'none',
  outfile: `${out}/readium.js`,
  loader: { '.css': 'text' },
  plugins: [stillSwipes],
})
await cp('node_modules/@readium/navigator/LICENSE', `${out}/LICENSE`)
await writeFile(`${out}/VERSION`, 'navigator 2.14.0, shared 2.8.0\n')
