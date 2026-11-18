import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile, access } from 'node:fs/promises'

const read = name => readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')

test('there is no Background picker: the controller is gone and nothing mounts #theme-backdrop', async () => {
  await assert.rejects(access(new URL('../static/backdrop-controller.js', import.meta.url)))
  for (const file of ['theme-controller.js', 'theme-settings-template.js', 'app.js']) {
    const source = await read(file)
    assert.doesNotMatch(source, /theme-backdrop|backdrop-controller|mountBackdrop/, file)
  }
})

test('the one automatic backdrop is the glow on Espresso and Kiln, and never on Paper or e-ink', async () => {
  const css = await read('backdrops.css')
  assert.doesNotMatch(css, /data-backdrop/, 'no per-choice selectors remain')
  for (const dropped of ['aurora', 'dawn', 'mist', 'spines']) assert.doesNotMatch(css, new RegExp(dropped), dropped)
  const rule = css.match(/([^{}]*)body::before\{[^}]*background:/)?.[1] || ''
  assert.match(rule, /data-palette=bobobooks/)
  assert.match(rule, /data-palette=kiln/)
  assert.doesNotMatch(rule, /paper|eink/)
  assert.doesNotMatch(css, /animation/, 'no motion in the backdrop')
})

test('pages no longer carry a data-backdrop attribute', async () => {
  for (const page of ['index.html', 'editor.html', 'import.html', '404.html']) assert.doesNotMatch(await read(page), /data-backdrop/, page)
})

test('the admin Sign-in background choice is gone from Settings', async () => {
  await assert.rejects(access(new URL('../static/login-look-settings.js', import.meta.url)))
  assert.doesNotMatch(await read('settings.js'), /login-look|loginLook|login_background/)
  assert.doesNotMatch(await read('login.css'), /data-login|spines/)
})
