import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import { chromium } from 'playwright'

test('warm filter tints the chapter, remains interactive and persists across reloads', async t => {
  const server = createServer(async (req, res) => {
    try {
      const name = new URL(req.url, 'http://localhost').pathname.replace(/^\/(?:assets\/)?/, '') || 'reader-readium.html'
      if (!/^[\w.-]+$/.test(name)) throw Error('invalid path')
      let body = await readFile(new URL(`../static/${name}`, import.meta.url), 'utf8')
      if (name === 'reader-readium.html') {
        body = body.replace(/<script(?![^>]*readium-warm-filter)[^>]*>[\s\S]*?<\/script>/g, '')
        res.setHeader('Content-Security-Policy', "script-src 'self'; style-src 'self' 'unsafe-inline'; frame-src 'self' data:")
      }
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
  const page = await browser.newPage()
  const mount = async () => {
    await page.goto(`http://127.0.0.1:${server.address().port}/`)
    await page.evaluate(() => {
      document.querySelector('#loading').hidden = true
      document.querySelector('#readium-host').innerHTML = `<iframe title="chapter" style="width:100%;height:100%" srcdoc="<body style='background:white;padding:100px'><button>chapter</button></body>"></iframe>`
      document.querySelector('#reader-settings').showModal()
      // The main reader script (which drives the settings tabs) is stripped from this
      // page on purpose, to test the warm-filter module in isolation; show its tab by hand.
      document.querySelector('#rs-panel-display').hidden = false
    })

  }
  await mount()
  const slider = page.getByRole('slider', { name: /warm/i })
  assert.equal(await slider.inputValue(), '0')
  assert.match(await page.locator('#reading-warmth-note').textContent(), /sepia.*images.*less vivid/i)
  assert.match(await page.locator('#reading-warmth-note').textContent(), /not a calibrated blue-light filter/i)
  await slider.fill('70')
  assert.match(await page.locator('#reading-warmth-status').textContent(), /saved in this browser/i)
  assert.match(await slider.getAttribute('aria-valuetext'), /70%/)
  assert.match(await page.locator('#readium-host').evaluate(el => getComputedStyle(el).filter), /sepia\(0\.7\)/)
  await page.locator('#reader-settings').evaluate(el => el.close())
  await page.frameLocator('iframe[title="chapter"]').getByRole('button', { name: 'chapter' }).evaluate(el => el.addEventListener('click', () => { el.dataset.clicked = '1' }))
  await page.frameLocator('iframe[title="chapter"]').getByRole('button', { name: 'chapter' }).click()
  assert.equal(await page.frameLocator('iframe[title="chapter"]').getByRole('button', { name: 'chapter' }).getAttribute('data-clicked'), '1')
  await page.reload()
  await mount()
  assert.equal(await slider.inputValue(), '70')
  await slider.focus()
  await page.keyboard.press('ArrowLeft')
  assert.equal(await slider.inputValue(), '69')

  const blocked = await browser.newPage()
  await blocked.addInitScript(() => {
    Storage.prototype.setItem = function () { throw new DOMException('blocked', 'SecurityError') }
  })
  await blocked.goto(`http://127.0.0.1:${server.address().port}/`)
  await blocked.locator('#reader-settings').evaluate(el => el.showModal())
  await blocked.locator('#rs-panel-display').evaluate(el => { el.hidden = false })
  await blocked.getByRole('slider', { name: /warm/i }).fill('60')
  assert.match(await blocked.locator('#reading-warmth-status').textContent(), /not saved/i)
  assert.match(await blocked.locator('#readium-host').evaluate(el => getComputedStyle(el).filter), /sepia\(0\.6\)/)
})
