import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { chromium } from 'playwright'

// Real menu module/CSS with isolated nested-scroll geometry; no personal data.
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true })
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('http://menus.test/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (/^\/static\/[\w-]+\.(js|css)$/.test(path)) {
        await route.fulfill({ contentType: path.endsWith('.css') ? 'text/css' : 'text/javascript',
          body: await readFile(new URL(`../internal/web${path}`, import.meta.url), 'utf8') })
      } else await route.fulfill({ contentType: 'text/html', body: `
        <link rel="stylesheet" href="/static/viewport-action-menus.css">
        <style>body{margin:20px}main{height:600px;overflow:auto}
        article{margin-top:300px}article>details>div{width:200px;height:100px}
        .tail{height:1200px}</style>
        <main><article class="audio-item"><details class="audio-more" id="active">
        <summary>Track options</summary><div><button>Favorite</button></div></details></article>
        <details class="audio-more" id="closed-audio"><summary>Other track</summary><div>Closed</div></details>
        <details class="book-more"><summary>Other book</summary><div>Closed</div></details>
        <div class="tail"></div></main>` })
    })
    await page.goto('http://menus.test/')
    await page.evaluate(async () => {
      const { installViewportActionMenus } = await import('/static/viewport-action-menus.js')
      const query = document.querySelectorAll.bind(document)
      window.menuQueries = []
      document.querySelectorAll = selector => {
        const result = query(selector)
        if (selector.includes('[open]')) window.menuQueries.push([...result].map(item => item.id))
        return result
      }
      window.disposeMenus = installViewportActionMenus({ document, window })
    })
    await page.locator('#active > summary').click()
    if (width > 680) {
      const panel = page.locator('#active > div')
      await page.waitForFunction(() => document.querySelector('#active').classList.contains('viewport-menu'))
      const before = await panel.boundingBox()
      await page.locator('main').evaluate(element => { element.scrollTop = 120 })
      await page.waitForFunction(top => Math.abs(document.querySelector('#active > div').getBoundingClientRect().top - (top - 120)) < 1, before.y)
      const queries = await page.evaluate(() => window.menuQueries)
      assert.ok(queries.length >= 2)
      assert.ok(queries.every(ids => ids.length === 1 && ids[0] === 'active'), 'closed menus must not be repositioning candidates')
      await page.locator('#active > summary').click()
      await page.waitForFunction(() => !document.querySelector('article').classList.contains('viewport-menu-owner'))
    } else {
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
      assert.equal(await page.locator('.viewport-menu').count(), 0, 'mobile menu layout remains unchanged')
    }
    await page.evaluate(() => window.disposeMenus())
    assert.equal(await page.locator('.viewport-menu-owner').count(), 0)
    assert.deepEqual(errors, [])
    console.log(`Action menu nested scrolling, candidate filtering and cleanup passed at ${width}px`)
    await page.close()
  }
} finally { await browser.close() }
