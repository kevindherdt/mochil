import {chromium} from 'playwright'
import {readFile} from 'node:fs/promises'
import assert from 'node:assert/strict'

const browser = await chromium.launch({executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless:true})
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({viewport:{width,height:900}})
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('http://wiki-render.test/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (/^\/[a-z-]+\.(js|css)$/.test(path)) {
        await route.fulfill({
          contentType: path.endsWith('.js') ? 'text/javascript; charset=utf-8' : 'text/css; charset=utf-8',
          body: await readFile(new URL(`../internal/web/static${path}`, import.meta.url), 'utf8'),
        })
      } else await route.fulfill({contentType:'text/html; charset=utf-8',body:`
        <meta charset="utf-8"><link rel="stylesheet" href="/vault.css">
        <style>:root{--text:#eee;--accent:#efab78;--bg:#202020;--line:#555}
        body{background:var(--bg);color:var(--text);font:16px sans-serif;margin:20px}
        #pane{height:700px;overflow:auto}article{max-width:72ch;margin:auto}
        pre{background:#303030;padding:12px}</style>
        <section class="vault-document" id="pane"><article></article></section>`})
    })
    await page.goto('http://wiki-render.test')
    await page.evaluate(async () => {
      const {createVaultMarkdownRenderer} = await import('/vault-markdown.js')
      const source = [
        '---', 'title: Café 日本語', `long: ${'x'.repeat(600)}`, '---', '# Note',
        '[[#Café 日本語|Jump to heading]]', '',
        Array(12).fill('Heading').join(' | '), Array(12).fill('---').join(' | '),
        Array(12).fill('LongContentValue').join(' | '), '',
        ...Array(60).fill('Paragraph.\n'), '## Café 日本語', 'Target text.',
      ].join('\n')
      document.querySelector('article').innerHTML = createVaultMarkdownRenderer([], 'vault')(source, 'Note.md')
    })
    const disclosure = page.locator('.vault-properties')
    assert.equal(await disclosure.evaluate(element => element.open), false)
    await page.locator('summary').focus()
    await page.keyboard.press('Enter')
    assert.equal(await disclosure.evaluate(element => element.open), true)
    assert(await page.locator('summary').evaluate(element => element.getBoundingClientRect().height) >= 44)
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
    await page.keyboard.press('Enter')
    assert.equal(await disclosure.evaluate(element => element.open), false)

    assert.equal(await page.locator('th[scope="col"]').count(), 12)
    await page.locator('.vault-table-scroll').focus()
    assert.equal(await page.locator('.vault-table-scroll').evaluate(element => element === document.activeElement), true)
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)

    const jumped = await page.evaluate(async () => {
      const {parseVaultReference, jumpToVaultHeading} = await import('/vault-heading.js')
      const link = document.querySelector('a[href^="#vault:"]')
      const reference = parseVaultReference(link.getAttribute('href'))
      const pane = document.querySelector('#pane')
      return {ok:jumpToVaultHeading(pane, reference.heading), text:document.activeElement.textContent, scrolled:pane.scrollTop > 0}
    })
    assert.deepEqual(jumped, {ok:true,text:'Café 日本語',scrolled:true})
    assert.deepEqual(errors, [])
    console.log(`Wiki properties/table/heading fixture passed at ${width}px`)
    await page.close()
  }
} finally {
  await browser.close()
}
