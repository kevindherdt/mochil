import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';

const browser = await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true});
try {
  for (const width of [1440,390]) {
    const page = await browser.newPage({viewport:{width,height:900}, hasTouch:width===390});
    const errors=[]; page.on('pageerror', error=>errors.push(error.message));
    await page.route('http://video.test/**', async route => {
      const path=new URL(route.request().url()).pathname;
      if (/^\/static\/[\w-]+\.(js|css)$/.test(path)) await route.fulfill({contentType:path.endsWith('.css')?'text/css':'text/javascript',body:await readFile(new URL(`../internal/web${path}`,import.meta.url),'utf8')});
      else if(path==='/') await route.fulfill({contentType:'text/html',body:'<!doctype html><link rel="stylesheet" href="/static/app.css"><link rel="stylesheet" href="/static/video-polish.css"><select id="video-layout"><option value="covers">covers</option><option value="list">list</option></select><details id="view-menu"></details><div id="status"></div><div id="collection-index"></div>'});
      else await route.abort();
    });
    await page.goto('http://video.test/');
    await page.evaluate(async()=>{
      const {createVideoLibrary}=await import('/static/video-library.js');
      window.calls=[];
      const episodes={1:[{id:'one',title:'First',episode_title:'A first episode with a long title',season:1,episode:1,overview:'A long readable description. '.repeat(25)+'End of description.',progress:{position_seconds:30,duration_seconds:100},stream_url:'/one'}, {id:'two',title:'Second',season:1,episode:2,overview:'',stream_url:'/two'}],2:[{id:'three',title:'Third',season:2,episode:1,overview:'Season two description',stream_url:'/three'}]};
      const requestJSON=async (url,options={})=>{window.calls.push({url,method:options.method||'GET'});if(url.startsWith('/api/video-shows')) return {data:[{name:'Show',source_name:'Show',count:3,seasons:[{number:1,count:2},{number:2,count:1}]}]}; if(url.startsWith('/api/videos?')) return {data:episodes[new URL(url,location.href).searchParams.get('season') || 1],response:{headers:new Headers()}};return {data:{effective:{}}};};
      window.controller=createVideoLibrary({document,$:s=>document.querySelector(s),escapeHTML:s=>String(s).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('"','&quot;'),requestJSON});
      document.querySelector('#video-index').hidden=false;
      await window.controller.load('fixture','',true,'tv');
    });
    await page.locator('.video-show>summary').click();
    await page.locator('.video-season>summary').first().click();
    const first=page.locator('.video-season').first();
    await first.locator('[data-episode-index="1"]').waitFor();
    const cards=first.locator('.video-episode-card');
    const boxes=await cards.evaluateAll(es=>es.map(e=>{const r=e.getBoundingClientRect();return {x:r.x,y:r.y,bottom:r.bottom,right:r.right}}));
    const overview=first.locator('.video-episode-overview');
    assert.equal(await overview.isVisible(),true);
    const before=await cards.first().evaluate(e=>e.getBoundingClientRect().top+scrollY);
    if(width===390) await cards.nth(1).evaluate(e=>e.focus({preventScroll:true})); else await first.locator('[data-episode-index="1"]').hover();
    const after=await cards.first().evaluate(e=>e.getBoundingClientRect().top+scrollY);
    console.log(`${width}px first card y: long=${before}, short=${after}`);
    assert.ok(Math.abs(before-after)<1, 'collapsed preview must not shift episode cards');
    await page.mouse.move(0,0);
    await first.locator('[data-episode-index="0"]').evaluate(e=>e.focus({preventScroll:true}));
    const toggle=overview.locator('[data-episode-toggle]');
    assert.equal(await toggle.getAttribute('aria-label'),'Show full description');
    if(width===390) await toggle.tap(); else {await toggle.evaluate(e=>e.focus({preventScroll:true})); await page.keyboard.press('Enter');}
    assert.equal(await toggle.getAttribute('aria-expanded'),'true');
    assert.equal(await overview.locator('[data-episode-description]').evaluate(e=>e.clientHeight===e.scrollHeight),true);
    if(width===390) await toggle.tap(); else await page.keyboard.press('Space');
    assert.equal(await toggle.getAttribute('aria-expanded'),'false');
    assert.equal(await page.locator('#video-player').evaluate(e=>e.open),false);
    assert.match(await overview.locator('[data-episode-description]').textContent(),/long readable/);
    assert.equal(await page.locator('.video-episode-play').count(),0);
    assert.equal(await cards.first().locator('.video-progress').count(),1);
    assert.equal(await cards.first().getByRole('button',{name:/Play/}).count(),1);
    assert.equal(await page.locator('#video-player').evaluate(e=>e.open),false);
    if(width===390) await first.locator('[data-episode-index="1"]').tap();
    else { await first.locator('[data-episode-index="1"]').focus(); await page.keyboard.press('Enter'); }
    assert.equal(await page.locator('#video-player').evaluate(e=>e.open),true);
    assert.equal(await page.evaluate(()=>window.calls.some(c=>c.method!=='GET')),false);
    await page.evaluate(async()=>{document.querySelector('#video-player').close(); await window.controller.load('fixture','',true,'movies');});
    const generic=page.locator('#videos>.video-item').first();
    assert.equal(await generic.locator('button').count(),1);
    assert.equal(await generic.locator('.video-progress').count(),1);
    if(width===390) await generic.locator('button').tap(); else await generic.locator('button').click();
    assert.equal(await page.locator('#video-player').evaluate(e=>e.open),true);
    assert.deepEqual(errors,[]);
    console.log(`Episode preview, expansion and direct playback passed at ${width}px`);
    await page.close();
  }
} finally {await browser.close()}
