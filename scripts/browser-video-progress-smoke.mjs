import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {chromium} from 'playwright'

// Real player DOM/event bindings with a synthetic media clock and transport.
// This checks integration, not codec decoding, streaming or visual acceptance.
const browser = await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true})
try {
  for (const width of [1440,390]) {
    const page = await browser.newPage({viewport:{width,height:900}})
    const errors=[]
    page.on('pageerror', error => errors.push(error.message))
    await page.route('http://video.test/**', async route => {
      const path=new URL(route.request().url()).pathname
      if (/^\/static\/[\w-]+\.js$/.test(path)) {
        await route.fulfill({contentType:'text/javascript',body:await readFile(new URL(`../internal/web${path}`,import.meta.url),'utf8')})
      } else if (path === '/') {
        await route.fulfill({contentType:'text/html',body:'<!doctype html><main></main>'})
      } else await route.abort()
    })
    await page.goto('http://video.test/')
    await page.evaluate(async () => {
      const {createVideoPlayer}=await import('/static/video-player.js')
      window.writes=[]
      const player=createVideoPlayer({document,$:s=>document.querySelector(s),escapeHTML:s=>s,
        requestJSON:async (_path,options={}) => {
          if (!options.method) return {data:{effective:{}}}
          window.writes.push(JSON.parse(options.body))
          if (window.writes.length === 1) throw new Error('offline fixture')
          return {data:{revision:window.writes.length+1}}
        },
      })
      const video=document.querySelector('video')
      for (const [key,value] of Object.entries({duration:200,currentTime:20,paused:false,ended:false})) {
        Object.defineProperty(video,key,{value,writable:true,configurable:true})
      }
      video.play=async()=>{}
      video.pause=()=>{video.paused=true;video.dispatchEvent(new Event('pause'))}
      video.load=()=>{}
      await player.open({library_id:'tv',id:'one',stream_url:'/fixture.mp4',progress:{revision:1}},'Fixture episode')
      video.dispatchEvent(new Event('seeking'))
      video.dispatchEvent(new Event('timeupdate'))
    })
    await page.waitForFunction(()=>window.writes.length===1)
    await page.evaluate(()=>document.querySelector('video').dispatchEvent(new Event('timeupdate')))
    await page.waitForFunction(()=>window.writes.length===2)
    assert.deepEqual(await page.evaluate(()=>window.writes.map(x=>x.explicit_seek)),[true,true])
    await page.locator('#video-close').click()
    await page.waitForFunction(()=>window.writes.length===3)
    assert.equal(await page.locator('#video-player').evaluate(e=>e.open),false)
    await page.evaluate(()=>document.querySelector('video').dispatchEvent(new Event('pause')))
    assert.equal(await page.evaluate(()=>window.writes.length),3)
    assert.equal(await page.evaluate(()=>window.writes[2].explicit_seek),false)
    assert.deepEqual(errors,[])
    console.log(`Video seek retry and close/pause bindings passed at ${width}px`)
    await page.close()
  }
} finally { await browser.close() }
