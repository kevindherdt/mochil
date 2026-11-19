import {chromium} from 'playwright'
import {readFile} from 'node:fs/promises'
import assert from 'node:assert/strict'

const browser=await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true})
try {
  for(const width of [1440,390]) {
    const page=await browser.newPage({viewport:{width,height:900}})
    const errors=[]
    page.on('pageerror',error=>errors.push(error.message))
    await page.route('http://tasks.test/**',async route=>{
      const path=new URL(route.request().url()).pathname
      if(/^\/[a-z-]+\.js$/.test(path)) await route.fulfill({contentType:'text/javascript; charset=utf-8',body:await readFile(new URL(`../internal/web/static${path}`,import.meta.url),'utf8')})
      else await route.fulfill({contentType:'text/html; charset=utf-8',body:'<section id="pane"></section>'})
    })
    await page.goto('http://tasks.test')
    for(const outcome of ['success','reload-failed','offline','committed','conflict','detached']) {
      await page.evaluate(async outcome=>{
        const {createVaultMarkdownRenderer}=await import('/vault-markdown.js')
        const {bindVaultTasks}=await import('/vault-task.js')
        const pane=document.querySelector('#pane'), content='- [ ] first\r\n- [ ] second'
        pane.innerHTML='<button data-vault-edit>Edit</button><article>'+createVaultMarkdownRenderer([],'vault')(content,'Note.md',{writable:true})+'</article>'
        window.calls=[];window.saved=0;window.current=true
        bindVaultTasks({pane,item:{id:'note',library_id:'vault'},content,revision:'r1',writable:true,
          isCurrent:()=>window.current,onSaved:async()=>{window.saved++;if(outcome==='reload-failed')throw new Error('Reload failed')},
          requestJSON:(url,options)=>{window.calls.push({url,body:JSON.parse(options.body)});return new Promise((resolve,reject)=>{window.finish=resolve;window.fail=reject})}})
      },outcome)
      await page.locator('input').first().check()
      assert.equal(await page.locator('input:disabled').count(),2)
      await page.locator('input').last().dispatchEvent('change')
      const requests=await page.evaluate(()=>window.calls)
      assert.equal(requests.length,1)
      assert.equal(requests[0].body.revision,'r1')
      assert.equal(requests[0].body.content,'- [x] first\r\n- [ ] second')
      await page.evaluate(outcome=>{
        if(outcome==='success' || outcome==='reload-failed') window.finish({})
        else if(outcome==='detached') {window.current=false;document.querySelector('#pane').innerHTML='New page';window.fail(new Error('old failure'))}
        else if(outcome==='committed') window.fail(Object.assign(new Error('Saved'),{status:500,code:'vault_index_refresh_failed',committed:true,indexRefreshed:false}))
        else if(outcome==='conflict') window.fail(Object.assign(new Error('Reload this page'),{status:409}))
        else window.fail(new Error('Offline'))
      },outcome)
      if(outcome==='detached') assert.equal(await page.locator('#pane').textContent(),'New page')
      else if(outcome==='success') assert.equal(await page.evaluate(()=>window.saved),1)
      else {
        assert.equal(await page.locator('input').first().isChecked(),outcome==='committed' || outcome==='reload-failed')
        assert.equal(await page.locator('input:disabled').count(),outcome==='offline'?0:2)
        if(outcome==='reload-failed') {
          assert.equal(await page.locator('[role="status"]').textContent(),'Saved. Reopen the page before changing another task.')
          await page.locator('input').first().dispatchEvent('change')
          assert.equal(await page.evaluate(()=>window.calls.length),1)
        }
      }
    }
    assert.deepEqual(errors,[])
    console.log(`Wiki task save/recovery fixture passed at ${width}px`)
    await page.close()
  }
} finally {await browser.close()}
