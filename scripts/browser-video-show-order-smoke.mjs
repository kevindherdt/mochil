import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';

// An opened show must become the top hero no matter which grid position it
// occupied (opening the right-column card used to leave it stranded below).
const browser = await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true});
try {
  for (const width of [1440,390]) {
    const page = await browser.newPage({viewport:{width,height:900},hasTouch:width===390});
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
      const shows=[{name:'Alpha Show',source_name:'Alpha Show',count:2,seasons:[{number:1,count:2}]},{name:'Beta Show',source_name:'Beta Show',count:2,seasons:[{number:1,count:2}]},{name:'Gamma Show',source_name:'Gamma Show',count:2,seasons:[{number:1,count:2}]}];
      const requestJSON=async (url,options={})=>{if(url.startsWith('/api/video-shows')) return {data:shows}; if(url.startsWith('/api/videos?')) return {data:[{id:'ep1',title:'Ep 1',season:1,episode:1,stream_url:'/one'}]};return {data:{effective:{}}};};
      window.controller=createVideoLibrary({document,$:s=>document.querySelector(s),escapeHTML:s=>String(s).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('"','&quot;'),requestJSON});
      document.querySelector('#video-index').hidden=false;
      await window.controller.load('fixture','',true,'tv');
    });
    const grid=page.locator('#videos.video-cover-grid');
    const shows=grid.locator('> .video-show');
    await shows.count()===3 && shows.nth(2).waitFor();

    // The second show (right of the first in a multi-column grid) opens.
    await shows.nth(1).locator('>summary').click({force:true});
    await page.waitForTimeout(50);
    const openBox=await shows.nth(1).boundingBox();
    const firstClosed=await shows.nth(0).boundingBox();
    const lastClosed=await shows.nth(2).boundingBox();
    console.log(`${width}px open.y=${openBox.y.toFixed(1)} firstClosed.y=${firstClosed.y.toFixed(1)} lastClosed.y=${lastClosed.y.toFixed(1)}`);
    assert.ok(openBox.y < firstClosed.y, `opened show must sit above closed shows (open.y=${openBox.y}, first.y=${firstClosed.y})`);
    assert.ok(openBox.y < lastClosed.y, 'opened show must sit above all closed shows');
    assert.ok(openBox.width >= firstClosed.width * 2 - 2, 'opened show spans the full grid width');
    // DOM order must be untouched (focus order, bindings, navigation rely on it).
    assert.deepEqual(await shows.evaluateAll(es=>es.map(e=>e.querySelector('.video-show-copy strong')?.textContent)),['Alpha Show','Beta Show','Gamma Show']);

    // Opening the third show adds a second hero; both open shows stay above
    // the only closed one.
    await shows.nth(2).locator('>summary').click({force:true});
    await page.waitForTimeout(50);
    const openBox2=await shows.nth(2).boundingBox();
    const onlyClosed=await shows.nth(0).boundingBox();
    assert.ok(openBox2.y < onlyClosed.y, 'newly opened show must sit above the closed show');
    assert.deepEqual(errors,[]);
    console.log(`Opened-show ordering passed at ${width}px`);
    await page.close();
  }
} finally {await browser.close()}
