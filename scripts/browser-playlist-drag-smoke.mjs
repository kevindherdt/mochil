import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {chromium} from 'playwright'

const browser = await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true})
try {
  const page = await browser.newPage({viewport:{width:1440,height:900}})
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  await page.route('http://playlist-drag.test/**', async route => {
    const path = new URL(route.request().url()).pathname
    if (/^\/static\/[\w-]+\.js$/.test(path)) {
      await route.fulfill({contentType:'text/javascript',body:await readFile(new URL(`../internal/web${path}`,import.meta.url),'utf8')})
    } else await route.fulfill({contentType:'text/html',body:`
      <style>.audio-item{height:60px;margin:8px;padding:8px;border:1px solid #999}</style>
      <main id="items"></main><p id="status"></p>`})
  })
  await page.goto('http://playlist-drag.test/')
  await page.evaluate(async () => {
    const {createMusicState} = await import('/static/music-state.js')
    const {createMusicDragBindings} = await import('/static/music-drag-bindings.js')
    const tracks = ['one','two','three','four'].map(id=>({id,library_id:'music'}))
    window.writes = []
    window.intents = 0
    const barrier = new Promise(resolve=>{window.releaseWrite=resolve})
    const container = document.querySelector('#items')
    let bindings
    const render = () => {
      container.innerHTML = `<section class="music-playlist-tracks">${music.playlists[0].items.map(item=>`<div class="audio-item" data-id="${item.id}">${item.id}</div>`).join('')}</section>`
      bindings?.playlistDrag.sync(true)
    }
    const music = createMusicState({key:item=>item.id,onChanged:render,requestJSON:async(path,options={})=>{
      if (options.method === 'PUT') {
        window.writes.push(JSON.parse(options.body).map(item=>item.item_id))
        if (window.writes.length === 1) await barrier
        return {data:null}
      }
      return {data:path.includes('playlists')?[{name:'Fixture',items:tracks}]:[]}
    }})
    await music.load()
    const edit = music.editPlaylist
    music.editPlaylist = (...args) => {window.intents++;return edit(...args)}
    bindings = createMusicDragBindings({container,music,selectedPlaylist:()=> 'Fixture',key:item=>item.id,render,status:text=>{document.querySelector('#status').textContent=text}})
    render()
  })
  const dragAfter = async (from, target) => {
    const destination = page.locator(`[data-id="${target}"]`)
    const box = await destination.boundingBox()
    await page.locator(`[data-id="${from}"]`).dragTo(destination,{targetPosition:{x:30,y:box.height-10}})
  }
  await dragAfter('one','three')
  await page.waitForFunction(()=>window.writes.length===1)
  await dragAfter('two','four')
  await page.waitForFunction(()=>window.intents===2)
  await page.evaluate(()=>window.releaseWrite())
  await page.waitForFunction(()=>document.querySelector('.audio-item')?.dataset.id==='three')
  assert.deepEqual(await page.evaluate(()=>window.writes),[['two','three','one','four'],['three','one','four','two']])
  assert.deepEqual(await page.locator('.audio-item').allTextContents(),['three','one','four','two'])
  assert.equal(await page.locator('[data-drop-position],.is-dragging').count(),0)
  assert.deepEqual(errors,[])
  console.log('Desktop playlist drag bindings preserve successive gestures across a delayed save')
} finally {await browser.close()}
