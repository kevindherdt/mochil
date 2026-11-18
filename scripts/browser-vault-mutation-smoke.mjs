import {chromium} from 'playwright'
import {readFile} from 'node:fs/promises'
import assert from 'node:assert/strict'

const browser=await chromium.launch({executablePath:process.env.CHROMIUM_PATH || '/usr/bin/chromium',headless:true})
try {
  for(const width of [1440,390]) {
    const page=await browser.newPage({viewport:{width,height:900}})
    const errors=[]
    page.on('pageerror',error=>errors.push(error.message))
    await page.route('http://vault.test/**',async route=>{
      const path=new URL(route.request().url()).pathname
      if(['/vault-create.js','/vault-editor.js','/vault-mutation-error.js','/http-json.js'].includes(path)) {
        await route.fulfill({contentType:'text/javascript',body:await readFile(new URL(`../internal/web/static${path}`,import.meta.url),'utf8')})
      } else await route.fulfill({contentType:'text/html',body:'<main id="host"></main>'})
    })
    await page.goto('http://vault.test')
    await page.evaluate(async()=>{
      const {mountVaultCreate}=await import('/vault-create.js')
      window.calls=0;window.completions=0
      mountVaultCreate({host:document.querySelector('#host'),type:'page',library:'test',
        requestJSON:()=>{window.calls++;return new Promise(resolve=>window.finish=resolve)},
        onCreated:()=>window.completions++,
      })
      const form=document.querySelector('form');form.querySelector('input').value='First'
      form.onsubmit({preventDefault(){}});form.onsubmit({preventDefault(){}})
    })
    assert.equal(await page.evaluate(()=>window.calls),1)
    assert.equal(await page.locator('input').evaluate(input=>input.readOnly),true)
    await page.locator('button[type="button"]').click()
    await page.evaluate(async()=>{window.finish({});await Promise.resolve()})
    assert.equal(await page.evaluate(()=>window.completions),0)

    await page.evaluate(async()=>{
      const {mountVaultEditor}=await import('/vault-editor.js')
      window.calls=0;window.completions=0
      mountVaultEditor({pane:document.querySelector('#host'),item:{id:'page',library_id:'test',title:'Page',path:'Page.md'},
        content:'Keep this draft',revision:'revision',escapeHTML:value=>value,
        requestJSON:()=>{window.calls++;return new Promise((resolve,reject)=>{window.finish=resolve;window.reject=reject})},
        onCancel:()=>{document.querySelector('#host').innerHTML='Other page'},onSaved:()=>window.completions++,
      })
      document.querySelector('[data-vault-save]').onclick();document.querySelector('[data-vault-save]').onclick()
    })
    assert.equal(await page.evaluate(()=>window.calls),1)
    assert.equal(await page.locator('textarea').evaluate(editor=>editor.readOnly),true)
    assert.equal(await page.locator('[data-vault-format]:enabled').count(),0)
    await page.evaluate(async()=>{window.reject(new Error('Offline'));await Promise.resolve()})
    assert.equal(await page.locator('textarea').inputValue(),'Keep this draft')
    assert.equal(await page.locator('textarea').evaluate(editor=>editor.readOnly),false)
    await page.locator('[data-vault-save]').click()
    await page.locator('[data-vault-cancel]').click()
    await page.evaluate(async()=>{window.finish({});await Promise.resolve()})
    assert.equal(await page.evaluate(()=>window.completions),0)
    assert.equal(await page.locator('#host').textContent(),'Other page')
    for (const type of ['page','folder','editor','reload']) {
      await page.evaluate(async(type)=>{
        const {requestJSON}=await import('/http-json.js')
        window.calls=0
        window.fetch=async()=>{
          window.calls++
          return new Response(JSON.stringify(type==='reload' ? {ok:true} : {
            error:'Your Wiki change was saved, but the library index could not refresh. Refresh the library before making another change; do not repeat this operation.',
            code:'vault_index_refresh_failed',committed:true,index_refreshed:false,
          }),{status:type==='reload' ? 200 : 500})
        }
        if(type==='editor' || type==='reload') {
          const {mountVaultEditor}=await import('/vault-editor.js')
          mountVaultEditor({pane:document.querySelector('#host'),item:{id:'page',library_id:'test',title:'Page',path:'Page.md'},
            content:'Saved text remains selectable',revision:'old-revision',escapeHTML:value=>value,requestJSON,
            onCancel:()=>{document.querySelector('#host').innerHTML='Closed'},
            onSaved:()=>{throw new Error('Reload failed')},
          })
          await document.querySelector('[data-vault-save]').onclick()
          await document.querySelector('[data-vault-save]').onclick()
        } else {
          document.querySelector('#host').innerHTML=''
          const {mountVaultCreate}=await import('/vault-create.js')
          mountVaultCreate({host:document.querySelector('#host'),type,library:'test',requestJSON,onCreated:()=>{throw new Error('Unexpected completion')}})
          const form=document.querySelector('form');form.querySelector('input').value='Created'
          await form.onsubmit({preventDefault(){}})
          await form.onsubmit({preventDefault(){}})
        }
      },type)
      assert.equal(await page.evaluate(()=>window.calls),1)
      assert.match(await page.locator('[role="status"]').textContent(),/do not repeat/)
      if(type==='editor' || type==='reload') {
        assert.equal(await page.locator('textarea').inputValue(),'Saved text remains selectable')
        assert.equal(await page.locator('textarea').evaluate(element=>element.readOnly),true)
        assert.equal(await page.locator('[data-vault-save]').isDisabled(),true)
        assert.equal(await page.locator('[data-vault-format]:enabled').count(),0)
        await page.locator('[data-vault-cancel]').click()
        assert.equal(await page.locator('#host').textContent(),'Closed')
      } else {
        assert.equal(await page.locator('input').evaluate(element=>element.readOnly),true)
        assert.equal(await page.locator('button[type="submit"]').isDisabled(),true)
        await page.getByRole('button',{name:'Close',exact:true}).click()
        assert.equal(await page.locator('form').count(),0)
      }
    }
    assert.deepEqual(errors,[])
    console.log(`Vault mutation ownership passed at ${width}px`)
    await page.close()
  }
} finally {await browser.close()}
