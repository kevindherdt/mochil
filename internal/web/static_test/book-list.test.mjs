import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const read = name => readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')

test('the book list stylesheet is loaded by the shell and reuses the shared card tokens', async () => {
  const [css, shell] = await Promise.all([read('book-list.css'), read('index.html')])
  assert.match(shell, /book-list\.css/)
  for (const token of ['--card-border', '--card-radius', '--card-bg']) assert.match(css, new RegExp(`var\\(${token}`))
})

test('series rows carry a cover for the list layout', async () => {
  assert.match(await read('series-catalogue.js'), /series-row-cover/)
})
