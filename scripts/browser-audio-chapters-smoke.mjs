import {chromium} from 'playwright'
import {readFile} from 'node:fs/promises'
import {existsSync} from 'node:fs'
import assert from 'node:assert/strict'

const executablePath=process.env.CHROMIUM_PATH || (existsSync('/usr/bin/chromium') ? '/usr/bin/chromium' : undefined)
// Optional disposable candidate: exercise its embedded assets, never personal data.
const assetOrigin=process.argv[2] ? new URL(process.argv[2]) : null
if(assetOrigin) {
  assert.equal(assetOrigin.hostname,'127.0.0.1')
  assert.equal(assetOrigin.protocol,'http:')
  assert(assetOrigin.port && assetOrigin.port!=='8099')
  assert.equal(assetOrigin.pathname,'/')
}
async function asset(path) {
  if(!assetOrigin)return readFile(new URL(`../internal/web/static${path}`,import.meta.url),'utf8')
  const response=await fetch(new URL(`/assets${path}`,assetOrigin),{redirect:'error',signal:AbortSignal.timeout(10000)})
  assert.equal(response.status,200,`Candidate asset ${path}`)
  return response.text()
}
const browser=await chromium.launch({executablePath,headless:true})
try {
  for(const width of [1440,390]) {
    const page=await browser.newPage({viewport:{width,height:900}})
    const errors=[]
    page.on('pageerror',error=>errors.push(error.message))
    await page.route('http://chapters.test/**',async route=>{
      const path=new URL(route.request().url()).pathname
      if(/^\/[a-z-]+\.(js|css)$/.test(path)) return route.fulfill({contentType:path.endsWith('.js')?'text/javascript':'text/css',body:await asset(path)})
      await route.fulfill({contentType:'text/html',body:`<!doctype html><html data-theme="dark" data-palette="mochil"><head><link rel="stylesheet" href="/app.css"><link rel="stylesheet" href="/theme-tokens.css"><link rel="stylesheet" href="/audio-player.css"></head><body><button id="outside">Library</button><script type="module">
        import {mountAudioPlayer} from '/audio-player-template.js';
        import {createAudiobookChapters} from '/audiobook-chapters.js';
        mountAudioPlayer(document);
        const player=document.querySelector('#audio-player'); player.hidden=false; player.classList.add('expanded');
        const item={library_kind:'audiobooks',chapters:Array.from({length:80},(_,i)=>({title:'Chapter '+(i+1),start_seconds:i*60,end_seconds:(i+1)*60}))};
        window.audio={currentTime:3600,play:()=>Promise.resolve()};
        window.controller=createAudiobookChapters({$:s=>document.querySelector(s),audio,player,getActive:()=>item,clock:s=>String(s),escapeHTML:s=>s,expandPlayer:()=>player.classList.add('expanded'),updateTimeline(){}});
        controller.openItem(item); window.ready=true;
      </script></body></html>`})
    })
    await page.goto('http://chapters.test/')
    await page.waitForFunction(()=>window.ready)
    const open=async()=>{
      await page.locator('.audio-player-more summary').click()
      await page.locator('#audio-chapters-toggle').focus()
      await page.keyboard.press('Enter')
    }
    await open()
    const opened=await page.evaluate(()=>{
      const panel=document.querySelector('#audio-chapters'), active=panel.querySelector('.active');
      const a=active.getBoundingClientRect(), p=document.querySelector('#audio-chapter-list').getBoundingClientRect();
      return {focused:document.activeElement===active,focus:document.activeElement.id,activeVisible:a.top>=p.top-1&&a.bottom<=p.bottom+1,scrollY}
    })
    assert.equal(opened.focused,true,JSON.stringify({opened,errors}))
    assert.equal(opened.activeVisible,true)
    assert.equal(opened.scrollY,0)
    await page.keyboard.press('Escape')
    assert.equal(await page.locator('#audio-chapters').evaluate(e=>e.hidden),true)
    assert.equal(await page.locator('.audio-player-more summary').evaluate(e=>e===document.activeElement),true)
    assert.equal(await page.locator('#audio-chapters-toggle').getAttribute('aria-expanded'),'false')
    await open()
    await page.locator('#audio-chapters-close').focus()
    await page.keyboard.press('Enter')
    assert.equal(await page.locator('.audio-player-more summary').evaluate(e=>e===document.activeElement),true)
    await page.evaluate(()=>{audio.currentTime=-1;controller.update()})
    await open()
    assert.equal(await page.evaluate(()=>document.activeElement.id),'audio-chapters-close')
    await page.locator('#outside').focus()
    await page.keyboard.press('Escape')
    assert.equal(await page.locator('#audio-chapters').evaluate(e=>e.hidden),false,'outside Escape is not intercepted')
    await page.evaluate(()=>controller.close())
    assert.equal(await page.evaluate(()=>document.activeElement.id),'outside','background close does not steal focus')
    assert.deepEqual(errors,[])
    console.log(`Chapter focus, reveal, close and scoped Escape checks passed at ${width}px`)
    await page.close()
  }
} finally {await browser.close()}
