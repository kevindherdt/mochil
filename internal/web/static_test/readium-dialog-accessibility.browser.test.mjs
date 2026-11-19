import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import { chromium } from 'playwright'

test('Readium dialogs expose their visible titles to assistive technology', async t => {
  const server = createServer(async (req, res) => {
    try {
      const name = new URL(req.url, 'http://localhost').pathname.replace(/^\/(?:assets\/)?/, '') || 'reader-readium.html'
      if (!/^[\w.-]+$/.test(name)) throw Error('invalid path')
      let body = await readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')
      if (name.endsWith('.html')) body = body.replace(/<script[^>]*>[\s\S]*?<\/script>/g, '')
      res.setHeader('Content-Type', name.endsWith('.css') ? 'text/css' : 'text/html')
      res.end(body)
    } catch { res.statusCode = 404; res.end() }
  }).listen(0, '127.0.0.1')
  await new Promise(resolve => server.once('listening', resolve))
  t.after(() => server.close())
  const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true, args: ['--no-sandbox'] })
  t.after(() => browser.close())
  for (const width of [320, 1365]) {
    await t.test(`${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`http://127.0.0.1:${server.address().port}/`)
        const cdp = await page.context().newCDPSession(page)
        for (const [id, title] of [['contents', 'Contents'], ['reader-settings', 'Settings'], ['book-search', 'Search']]) {
          await page.locator(`#${id}`).evaluate(dialog => dialog.showModal())
          const tree = await cdp.send('Accessibility.getFullAXTree')
          const dialogs = tree.nodes.filter(node => node.role?.value === 'dialog' && !node.ignored)
          assert.equal(dialogs.length, 1)
          assert.equal(dialogs[0].name?.value, title, `${id} should announce its title`)
          await page.keyboard.press('Escape')
          assert.equal(await page.locator(`#${id}`).evaluate(dialog => dialog.open), false)
        }
      } finally { await page.close() }
    })
  }
})
