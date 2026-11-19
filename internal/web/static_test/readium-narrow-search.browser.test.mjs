import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import { chromium } from 'playwright'

test('Readium narrow settings and search excerpts', async t => {
  const server = createServer(async (req, res) => {
    try {
      const name = new URL(req.url, 'http://localhost').pathname.replace(/^\/(?:assets\/)?/, '') || 'reader-readium.html'
      if (!/^[\w.-]+$/.test(name)) throw Error('invalid path')
      let body = await readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')
      if (name.endsWith('.html')) body = body.replace(/<script[^>]*>[\s\S]*?<\/script>/g, '')
      res.setHeader('Content-Type', name.endsWith('.js') ? 'text/javascript' : name.endsWith('.css') ? 'text/css' : 'text/html')
      res.end(body)
    } catch { res.statusCode = 404; res.end() }
  }).listen(0, '127.0.0.1')
  await new Promise(resolve => server.once('listening', resolve))
  t.after(() => server.close())
  let browser
  try { browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true, args: ['--no-sandbox'] }) }
  catch (error) { t.skip(`Chromium unavailable: ${error.message}`); return }
  t.after(() => browser.close())
  const page = await browser.newPage({ viewport: { width: 320, height: 568 } })
  await page.goto(`http://127.0.0.1:${server.address().port}/`)

  await t.test('all settings controls fit the dialog at 320px', async () => {
    const overflow = await page.evaluate(() => {
      document.querySelector('#loading').hidden = true
      document.querySelector('#narration-settings').hidden = false
      const dialog = document.querySelector('#reader-settings')
      dialog.showModal()
      const bounds = dialog.getBoundingClientRect()
      // The tab strip scrolls horizontally on its own (like the app Settings nav does on
      // narrow screens), so its buttons may extend past the dialog; that is by design.
      return [...dialog.querySelectorAll('button, select, input, .rs-note')].filter(el => {
        if (el.closest('.rs-tabs')) return false
        const r = el.getBoundingClientRect()
        return r.width && (r.left < bounds.left - 1 || r.right > bounds.right + 1)
      }).map(el => el.id || el.className || el.tagName)
    })
    assert.deepEqual(overflow, [])
    assert.equal(await page.locator('#reader-settings').evaluate(el => el.scrollWidth <= el.clientWidth), true)
    if (process.env.READIUM_SCREENSHOT) await page.screenshot({ path: process.env.READIUM_SCREENSHOT })
    await page.locator('#size-up').click()
    await page.locator('#reading-font').selectOption('serif')
    assert.equal(await page.locator('#reading-font').inputValue(), 'serif')
  })

  await t.test('block boundaries separate excerpts but inline fragments stay joined', async () => {
    const result = await page.evaluate(async () => {
      const { findInDocument } = await import('/assets/readium-search.js')
      const doc = new DOMParser().parseFromString('<body><h1>Sample</h1><p>Test <em>book</em> and <span>un</span><b>broken</b>.</p></body>', 'text/html')
      return ['Test', 'Sample Test', 'unbroken'].map(q => findInDocument(doc, q)[0]?.excerpt)
    })
    assert.equal(result[0].pre + result[0].match + result[0].post, 'Sample Test book and unbroken.')
    assert.equal(result[1].match, 'Sample Test')
    assert.equal(result[2].match, 'unbroken')
  })
})
