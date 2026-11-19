import assert from 'node:assert/strict'
import http from 'node:http'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { chromium } from 'playwright'

const root = path.resolve('internal/web/static')
const server = http.createServer(async (req, res) => {
  let url = new URL(req.url, 'http://localhost').pathname
  if (url === '/reader/books/1') url = '/assets/reader-readium.html'
  if (url.startsWith('/assets/')) {
    const file = path.join(root, url.slice(8))
    try {
      const data = await readFile(file)
      res.setHeader('Content-Type', file.endsWith('.js') ? 'text/javascript' : file.endsWith('.css') ? 'text/css' : file.endsWith('.html') ? 'text/html' : 'application/octet-stream')
      return res.end(data)
    } catch { res.statusCode = 404; return res.end() }
  }
  res.setHeader('Content-Type', 'application/json')
  if (url.endsWith('/manifest.json')) return res.end(JSON.stringify({ metadata: { '@type': 'http://schema.org/Book', title: 'Synthetic', identifier: 'test', language: 'en' }, links: [{ href: 'positions.json', type: 'application/vnd.readium.position-list+json', rel: ['http://readium.org/positions'] }], readingOrder: [{ href: 'chapter.xhtml', type: 'application/xhtml+xml' }] }))
  if (url.endsWith('/positions.json')) return res.end(JSON.stringify({ total: 1, positions: [{ href: 'chapter.xhtml', type: 'application/xhtml+xml', locations: { position: 1, progression: 0, totalProgression: 0 } }] }))
  if (url.endsWith('/chapter.xhtml')) {
    res.setHeader('Content-Type', 'application/xhtml+xml')
    return res.end(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>Test</title></head><body><p id="probe">${'A simple sample text for testing the margins and measuring rendered layout. '.repeat(200)}</p></body></html>`)
  }
  if (url.startsWith('/api/reading/')) return res.end('{}')
  res.statusCode = 404; res.end()
})
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', args: ['--no-sandbox'] })
try {
  for (const [width, flow, columns] of [[1440, 'paginated', '1'], [390, 'paginated', '1'], [1440, 'paginated', '2'], [1440, 'paginated', '0'], [1440, 'scrolled', '1'], [390, 'scrolled', '1']]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    await page.goto(`http://127.0.0.1:${server.address().port}/reader/books/1`)
    await page.locator('#loading').waitFor({ state: 'hidden', timeout: 15000 })
    await page.frameLocator('#readium-host iframe').locator('#probe').waitFor({ timeout: 15000 })
    await page.locator('#settings-open').evaluate(el => el.click())
    await page.locator('[data-rs-tab="layout"]').evaluate(el => el.click())
    await page.locator('#reading-flow').selectOption(flow)
    if (flow === 'paginated') await page.locator('#reading-columns').selectOption(columns)
    const measure = async margin => {
      await page.locator('#reading-margin').selectOption(margin)
      await page.waitForTimeout(700)
      const box = await page.locator('#readium-host iframe').first().boundingBox()
      const rect = await page.frameLocator('#readium-host iframe').locator('#probe').evaluate(el => { const range = document.createRange(); range.selectNodeContents(el); const r = range.getClientRects()[0]; return { x: r.x, width: r.width } })
      return { x: box.x + rect.x, width: rect.width }
    }
    const sizes = []
    for (const margin of ['16', '24', '40', '64']) sizes.push(await measure(margin))
    console.log({ width, flow, columns, sizes })
    for (let i = 1; i < sizes.length; i++) {
      assert.ok(sizes[i].x > sizes[i - 1].x + 1, 'larger margins move rendered text inward')
    }
    assert.ok(sizes[3].width < sizes[0].width - 10, 'wide margins narrow rendered text')
    await page.close()
  }
} finally { await browser.close(); server.close() }
