import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { chromium } from 'playwright'

// Synthetic scroll geometry, actual navigation modules. Not a visual design audit.
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true })
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('http://navigation.test/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (/^\/static\/[\w-]+\.js$/.test(path)) {
        await route.fulfill({ contentType: 'text/javascript', body: await readFile(new URL(`../internal/web${path}`, import.meta.url), 'utf8') })
      } else {
        await route.fulfill({ contentType: 'text/html', body: `
          <style>:root{--catalog-sticky-top:60px}body{margin:0;padding-top:60px;padding-bottom:1000px}
          nav{position:fixed;top:0;left:0;right:0;background:white;z-index:2}section{height:600px;scroll-margin-top:60px}h2{margin:0}</style>
          <nav id="alphabet"></nav><main>
          <section><h2>The Album</h2></section><section><h2>An Record</h2></section>
          <section><h2>The\tAlbum</h2></section><section><h2>ßound</h2></section></main>` })
      }
    })
    await page.goto('http://navigation.test/')
    await page.evaluate(async () => {
      const { createMediaSectionNavigator } = await import('/static/media-section-navigator.js')
      window.navigatorFixture = createMediaSectionNavigator(document.querySelector('#alphabet'))
      window.navigatorFixture.show([...document.querySelectorAll('section')].map(element => ({ element, label: element.querySelector('h2').textContent })))
    })
    const active = page.locator('#alphabet [aria-pressed="true"]')
    await page.waitForFunction(() => document.querySelector('#alphabet [aria-pressed="true"]')?.dataset.bucket === 'A')
    for (const bucket of ['R', 'T', '#', 'A']) {
      await page.locator(`#alphabet [data-bucket="${bucket}"]`).click()
      await page.waitForFunction(expected => document.querySelector('#alphabet [aria-pressed="true"]')?.dataset.bucket === expected, bucket)
      assert.equal(await active.count(), 1)
    }
    // Native scrolling, rather than a letter click, must update the active marker.
    await page.evaluate(() => window.scrollTo(0, 1200))
    await page.waitForFunction(() => document.querySelector('#alphabet [aria-pressed="true"]')?.dataset.bucket === 'T')
    assert.equal(await page.locator('#alphabet [data-bucket="S"]').isDisabled(), true)
    await page.evaluate(() => window.navigatorFixture.hide())
    assert.equal(await page.locator('#alphabet').isVisible(), false)
    assert.deepEqual(errors, [])
    console.log(`Navigation module loading, bucket jumps, scroll selection and hide passed at ${width}px`)
    await page.close()
  }
} finally { await browser.close() }
