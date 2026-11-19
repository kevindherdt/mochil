import {chromium} from 'playwright'
import assert from 'node:assert/strict'

// Only run against a disposable server containing the documented Fixture notes vault.
const origin = new URL(process.argv[2] || 'http://127.0.0.1:18102')
assert.equal(origin.hostname, '127.0.0.1')
assert.equal(origin.protocol, 'http:')
assert(origin.port && origin.port !== '8099', 'Never target the personal server')
const browser = await chromium.launch({executablePath: process.env.CHROMIUM_PATH || '/usr/bin/chromium', headless:true})
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({viewport:{width,height:900}})
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.goto(origin.origin)
    await page.selectOption('#filters', {label:'Fixture notes'})
    await page.locator('.vault-page').filter({hasText:/^Start$/}).click()
    const article = page.locator('.vault-pane article')
    await article.locator('.vault-properties').waitFor()
    assert.equal(await article.locator('.vault-properties').evaluate(e => e.open), false)
    assert.equal(await article.locator('th').count(), 2)
    const task = article.locator('input[data-vault-task]').first()
    const previous = await task.isChecked()
    const saved = page.waitForResponse(r => r.request().method() === 'PUT' && r.url().includes('/api/vaults/'))
    await task.setChecked(!previous)
    assert.equal((await saved).status(), 200)
    await page.waitForFunction(expected => {
      const input = document.querySelector('.vault-pane input[data-vault-task]')
      return input && !input.disabled && input.checked === expected
    }, !previous)
    await page.reload()
    await page.selectOption('#filters', {label:'Fixture notes'})
    const opener = page.getByRole('button', {name:'Open vault sidebar',exact:true})
    if (await opener.isVisible()) await opener.click()
    await page.locator('.vault-page').filter({hasText:/^Start$/}).click()
    await task.waitFor()
    assert.equal(await task.isChecked(), !previous)
    await article.getByRole('link', {name:'Open details'}).click()
    await page.waitForFunction(() => document.activeElement?.textContent === 'Details')
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
    assert.deepEqual(errors, [])
    console.log(`Real-server wiki task persistence, properties, table and heading navigation passed at ${width}px`)
    await page.close()
  }
} finally {
  await browser.close()
}
