import assert from 'node:assert/strict'
import { chromium } from 'playwright'

// Phone-sized regressions that unit tests cannot see: pages that jump, taps that land on the wrong control, panels off screen.
// Needs a running server with a music and a book library (BOBOBOOKS_BASE_URL, OPDSHELF_USERNAME/OPDSHELF_PASSWORD).
const base = process.env.BOBOBOOKS_BASE_URL || 'http://127.0.0.1:8099'
const credentials = process.env.OPDSHELF_USERNAME ? { username: process.env.OPDSHELF_USERNAME, password: process.env.OPDSHELF_PASSWORD || '' } : undefined
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless: true, args: ['--autoplay-policy=no-user-gesture-required'] })

async function phone(layout) {
  const context = await browser.newContext({ viewport: { width: 412, height: 900 }, isMobile: true, hasTouch: true, httpCredentials: credentials })
  if (credentials) { // the app signs in with a form; the cookie then belongs to this whole context
    const signIn = await context.newPage()
    await signIn.goto(`${base}/login`)
    await signIn.fill('input[name=username]', credentials.username)
    await signIn.fill('input[name=password]', credentials.password)
    await Promise.all([signIn.waitForNavigation(), signIn.click('button[type=submit]')])
    await signIn.close()
  }
  const page = await context.newPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  await page.addInitScript(value => { localStorage.setItem('bobobooks-music-layout', value); localStorage.setItem('bobobooks-theme-mode', 'dark') }, layout)
  return { page, errors, context }
}

async function openTracks(page) {
  await page.goto(`${base}/#/library/music`)
  await page.locator('.destination-rail button', { hasText: 'Tracks' }).first().click()
  await page.waitForSelector('#audio-items>.audio-item')
  // Scroll the way a person does, in steps, so rows pass through the screen and are measured; load a few more pages on the way.
  for (let y = 0; y < 9000; y += 450) { await page.evaluate(top => window.scrollTo(0, top), y); await page.waitForTimeout(60) }
  await page.waitForTimeout(1500)
  await page.waitForTimeout(800)
}

for (const layout of ['list', 'covers']) {
  const { page, errors, context } = await phone(layout)
  await openTracks(page)
  const rows = page.locator('#audio-items>.audio-item')
  assert.ok(await rows.count() > 100, `${layout}: need at least 100 tracks to test scrolling`)
  // The list may be rebuilt after a tap, so the same track is found again by its text.
  const mark = await page.evaluate(() => { const items = [...document.querySelectorAll('#audio-items>.audio-item')]; return items.find(item => item.getBoundingClientRect().top > 150).textContent })
  const anchor = () => page.evaluate(text => { const row = [...document.querySelectorAll('#audio-items>.audio-item')].find(item => item.textContent === text); return row ? [Math.round(row.getBoundingClientRect().top), Math.round(scrollY)] : ['row missing'] }, mark)
  for (const tap of [0, 1]) {
    const before = await anchor() // the second tap happens with the mini player already open
    const box = await page.evaluate(() => { const row = [...document.querySelectorAll('#audio-items .audio-item .audio-open')].find(item => { const r = item.getBoundingClientRect(); return r.top > 200 && r.bottom < 760 }); const r = row.getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.top + r.height / 2 } })
    await page.touchscreen.tap(box.x, box.y)
    for (let i = 0; i < 6; i++) { await page.waitForTimeout(300); assert.deepEqual(await anchor(), before, `${layout}: the page moved after tap ${tap + 1} (was ${before}, now ${await anchor()})`) }
  }
  assert.deepEqual(errors, [], `${layout}: page errors`)
  await context.close()
}

{ // The options sheet must be fully on screen.
  const { page, context } = await phone('covers')
  await openTracks(page)
  await page.locator('.audio-view-menu > summary').first().click({ force: true })
  await page.waitForTimeout(500)
  const panel = await page.locator('.audio-view-menu[open] > div').boundingBox()
  assert.ok(panel.y >= 0 && panel.y + panel.height <= 900, `options sheet off screen: ${JSON.stringify(panel)}`)
  await context.close()
}

{ // Inside a series: the back arrow takes the tap, and there is no A-Z strip.
  const { page, context } = await phone('covers')
  await page.goto(`${base}/#/library/books`)
  await page.waitForSelector('.series-card .cover-frame')
  await page.locator('.series-card .cover-frame').nth(2).tap()
  await page.waitForSelector('.series-hub .back-button')
  const top = await page.evaluate(() => { const r = document.querySelector('.series-hub .back-button').getBoundingClientRect(); return document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2).closest('button')?.className })
  assert.equal(top, 'back-button', `the back arrow is covered by: ${top}`)
  const bar = await page.evaluate(() => ({ bar: document.querySelector('.masthead').getBoundingClientRect().bottom, rail: document.querySelector('.shell-context .destination-rail').getBoundingClientRect().bottom }))
  assert.ok(bar.rail <= bar.bar - 2, `the tab row spills over the bottom of the top bar (${bar.rail} vs ${bar.bar})`)
  assert.equal(await page.evaluate(() => getComputedStyle(document.querySelector('#alphabet')).display), 'none', 'A-Z strip should be hidden inside a series')
  await context.close()
}

{ // A refresh keeps the tab you were on (the address carries it).
  const { page, context } = await phone('covers')
  await page.goto(`${base}/#/library/books`)
  await page.waitForSelector('.series-card .cover-frame')
  await page.locator('[data-publication-destination=authors]').tap()
  await page.waitForFunction(() => location.hash === '#/library/books/authors')
  await page.reload()
  await page.waitForFunction(() => location.hash === '#/library/books/authors')
  await page.waitForSelector('.author-card')
  await context.close()
}

{ // Discover is the first tab: what is in progress, then what is new, without the A-Z strip; icon tabs keep their names for screen readers.
  const { page, context } = await phone('covers')
  await page.goto(`${base}/#/library/books`)
  await page.waitForSelector('.series-card .cover-frame')
  const labels = await page.evaluate(() => [...document.querySelectorAll('.publication-destinations button')].map(button => button.getAttribute('aria-label') || button.textContent.trim()))
  assert.deepEqual(labels, ['Discover', 'Series', 'Authors', 'All', 'Collections'])
  assert.equal(await page.evaluate(() => !document.querySelector('#reading-strip')), true, 'a book library no longer repeats the Continue reading shelf that Home already shows')
  await page.locator('[data-publication-destination=discover]').tap()
  await page.waitForFunction(() => location.hash === '#/library/books/discover')
  await page.waitForSelector('#book-catalog:not([hidden])')
  assert.equal(await page.evaluate(() => getComputedStyle(document.querySelector('#alphabet')).display), 'none', 'A-Z strip should be hidden on Discover')
  const names = await page.evaluate(() => [...document.querySelectorAll('.publication-destinations button')].filter(button => button.querySelector('span') && getComputedStyle(button.querySelector('span')).display !== 'none').map(button => button.getAttribute('aria-label')))
  assert.deepEqual(names, ['Discover'], 'only the current tab shows its name on a phone')
  const switcher = await page.evaluate(() => ({ width: document.querySelector('.shelf-switcher').getBoundingClientRect().width, icon: !!document.querySelector('.library-menu-mark svg'), name: getComputedStyle(document.querySelector('.library-menu-button span')).display }))
  assert.ok(switcher.width <= 80 && switcher.icon && switcher.name === 'none', `the library switcher is an icon on a phone (${JSON.stringify(switcher)})`)
  await page.locator('.library-menu-button').tap()
  const menu = await page.evaluate(() => ({ items: document.querySelectorAll('.library-menu-list li').length, icons: document.querySelectorAll('.library-menu-list li svg.kind-icon').length, last: document.querySelector('.library-menu-list li:last-child').textContent }))
  assert.ok(menu.items >= 3 && menu.icons === menu.items, `every menu entry has an icon (${menu.icons} of ${menu.items})`)
  assert.equal(menu.last, 'Settings')
  await context.close()
}

await browser.close()
console.log('browser stability smoke passed')
