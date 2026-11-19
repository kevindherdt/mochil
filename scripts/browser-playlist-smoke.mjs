import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { chromium } from 'playwright'

// Real browser forms and feature modules, isolated from personal libraries.
const executablePath = process.env.CHROMIUM_PATH || (existsSync('/usr/bin/chromium') ? '/usr/bin/chromium' : undefined)
const browser = await chromium.launch({ executablePath, headless: true })
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('http://playlist.test/**', async route => {
      const path = new URL(route.request().url()).pathname
      if (path.startsWith('/static/') && /^\/static\/[\w-]+\.js$/.test(path)) {
        const body = await readFile(new URL(`../internal/web${path}`, import.meta.url), 'utf8')
        await route.fulfill({ contentType: 'text/javascript', body })
      } else {
        await route.fulfill({ contentType: 'text/html', body: '<form id="playlist-create"><input id="playlist-name"><button>Save</button></form><button id="playlist-create-cancel">Cancel</button><p id="playlist-status"></p><main id="items"></main>' })
      }
    })
    await page.goto('http://playlist.test/')
    await page.evaluate(async () => {
      const { createMusicPlaylistController } = await import('/static/music-playlist-controller.js')
      const { musicPlaylistDetailHTML } = await import('/static/music-playlist-view.js')
      const playlist = { name: 'Night', items: [] }
      const container = document.querySelector('#items')
      const render = () => { container.innerHTML = musicPlaylistDetailHTML(playlist, () => '', value => value) }
      const music = {
        playlists: [playlist],
        renamePlaylist: () => new Promise((_, reject) => { window.rejectRename = reject }),
      }
      window.controller = createMusicPlaylistController({
        $: selector => document.querySelector(selector), container, music,
        key: item => item.id, render, open() {},
      })
      window.controller.setSelected('Night')
      render()
    })
    await page.locator('[data-playlist-rename-form]').evaluate(form => { form.hidden = false })
    await page.locator('[name="name"]').fill('Renamed')
    await page.locator('[data-playlist-rename-form]').evaluate(form => form.requestSubmit())
    await page.evaluate(() => window.rejectRename(new Error('Offline — try again')))
    await page.waitForFunction(() => document.querySelector('[data-playlist-rename-form] p')?.textContent === 'Offline — try again')
    assert.equal(await page.locator('[data-playlist-rename-form]').isVisible(), true)
    assert.equal(await page.locator('[name="name"]').inputValue(), 'Renamed', 'failed rename must preserve user input')
    await page.locator('[data-playlist-rename-form]').evaluate(form => form.requestSubmit())
    await page.evaluate(() => { window.controller.reset(); window.rejectRename(new Error('Late failure')) })
    await page.waitForTimeout(0)
    assert.equal(await page.evaluate(() => window.controller.selected), '')
    await page.evaluate(async () => {
      const {createMusicState} = await import('/static/music-state.js')
      const {createMusicPlaylistController} = await import('/static/music-playlist-controller.js')
      const {musicPlaylistDetailHTML, musicPlaylistOverviewHTML} = await import('/static/music-playlist-view.js')
      const container = document.querySelector('#items')
      let controller
      const music = createMusicState({key:item=>item.id,onChanged:()=>render(),requestJSON:async(path,options={})=>{
        if(options.method)return{data:null}
        if(path==='/api/audio-favorites')return{data:[]}
        if(path.includes('?view=summary'))return{data:{version:1,playlists:[{name:'Saved',item_count:200,artwork_urls:[]}]}}
        return new Promise(resolve=>{window.resolveDetail=resolve})
      }})
      const render = () => {
        const list = music.playlists.find(value=>value.name===controller?.selected)
        container.innerHTML = list ? musicPlaylistDetailHTML(list,item=>`<p>${item.title}</p>`,String) : musicPlaylistOverviewHTML(music.playlists,String)
      }
      controller = createMusicPlaylistController({$:selector=>document.querySelector(selector),container,music,key:item=>item.id,render,open:item=>{window.playedID=item.id}})
      container.onclick = event=>controller.handleClick(event)
      await music.load()
      render()
    })
    await page.locator('[data-open-playlist]').click()
    assert.equal(await page.locator('[role="status"]').textContent(), 'Loading playlist…')
    assert.equal(await page.locator('[data-play-playlist]').count(), 0)
    await page.evaluate(()=>window.resolveDetail({data:{name:'Saved',items:[{id:'one',library_id:'music',title:'First track'}]}}))
    await page.locator('[data-play-playlist]').click()
    assert.equal(await page.evaluate(()=>window.playedID),'one')
    await page.evaluate(async () => {
      const {mountAudioPlayer} = await import('/static/audio-player-template.js')
      const {createMusicPlayerQueue} = await import('/static/audio-player-queue.js')
      mountAudioPlayer(document)
      document.querySelector('#audio-player').hidden = false
      const track = {id:'queue-one',title:'Queued track',library_kind:'music'}
      const music = {loadErrors:{queue:'Could not save <unsafe text>'},retryQueueSave(){
        window.queueRetryCount=(window.queueRetryCount||0)+1
        music.loadErrors={}
        queue.sync([track],track)
      }}
      const queue = createMusicPlayerQueue({$:selector=>document.querySelector(selector),music,open(){},key:item=>item.id,escapeHTML:value=>String(value).replaceAll('<','&lt;').replaceAll('>','&gt;')})
      queue.sync([track],track)
      document.querySelector('#audio-queue-toggle').click()
    })
    assert.equal(await page.locator('#audio-queue-items [role="status"]').textContent(),'Could not save <unsafe text>')
    assert.equal(await page.locator('#audio-queue-items unsafe').count(),0)
    await page.locator('[data-queue-retry]').click()
    assert.equal(await page.evaluate(()=>window.queueRetryCount),1)
    assert.equal(await page.locator('[data-queue-retry]').count(),0)
    assert.equal(await page.evaluate(()=>document.activeElement.id),'audio-queue-close')
    assert.deepEqual(errors, [])
    console.log(`Playlist form checks passed at ${width}px`)
    await page.close()
  }
} finally {
  await browser.close()
}
