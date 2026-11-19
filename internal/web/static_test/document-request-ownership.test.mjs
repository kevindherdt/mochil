import test from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import vm from 'node:vm'
import {isVaultPage, vaultPageEntries} from '../static/vault-pages.js'
import {createVaultPageResolver} from '../static/vault-page-resolver.js'
import {parseVaultReference, jumpToVaultHeading} from '../static/vault-heading.js'

const source = readFileSync(new URL('../static/document-library.js', import.meta.url), 'utf8')
  .replace(/^import .*;\n/gm, '').replace('export function', 'function')

function fixture() {
  const nodes = new Map()
  const $ = key => {
    if (!nodes.has(key)) nodes.set(key, {value:'title', innerHTML:'', textContent:'', hidden:false,
      classList:{toggle(){},remove(){},add(){}}, setAttribute(){}, dataset:{}, children:[],
      querySelector(){return null}, querySelectorAll(){return []}})
    return nodes.get(key)
  }
  const pending = []
  const context = vm.createContext({AbortController, URLSearchParams, Set, Map,
    isVaultPage, vaultPageEntries, createVaultPageResolver, parseVaultReference, jumpToVaultHeading,
    localStorage:{getItem(){return null},setItem(){}}, matchMedia:()=>({matches:false}),
    CSS:{escape:String}, renderMarkdown:String, bindCodeCopy(){}, mountDocumentLibrary(){},
    createVaultMarkdownRenderer:()=>String, vaultHomeHTML:(entries,o)=>'<p class="vault-empty">'+o.message+'</p>',
    bindVaultTasks(){}, vaultNoteHTML:()=>'', mountVaultOutline(){}, dropRepeatedTitle(){}, mountBacklinks(){}, mountVaultExtras:()=>({openSwitcher(){},showTag(){},noteOpened(){}}), graphButtonHTML:"", mountVaultSwitcher:()=>({open(){}}), newPagePath:v=>v,
    mountVaultCreate(){}, mountVaultEditor(){}, launchVaultEditor(){}, createVaultTree:()=>()=>'',
    createContinuousPager:()=>({refresh(){}}), createBoundedWindow:()=>({reset(){}}),
    document:{body:{classList:{toggle(){}}}}, $, escapeHTML:String,
    requestJSON:(url)=>new Promise((resolve,reject)=>pending.push({url,resolve,reject})),
  })
  vm.runInContext(source + '\nthis.controller=createDocumentLibrary({document,$,escapeHTML,requestJSON})', context)
  const resolve = (request, data=[], headers={}) => request.resolve({data,response:{headers:{get(name){return headers[name]??null}}}})
  return {controller:context.controller, pending, $, resolve}
}

test('empty writable vaults retain workspace and first-page creation controls', async () => {
  const f=fixture()
  const loading=f.controller.load('vault','',true,'vault')
  f.resolve(f.pending[0],[],{'X-Vault-Writable':'1'}); await loading
  const html=f.$('#documents').innerHTML
  assert.match(html,/class="vault-workspace"/)
  assert.match(html,/data-vault-create="page"/)
  assert.match(html,/data-vault-create="folder"/)
  assert.match(html,/No pages yet\./)
})

test('empty search keeps the sidebar but readonly vaults never gain write actions', async () => {
  const f=fixture()
  const loading=f.controller.load('vault','missing',true,'vault')
  f.resolve(f.pending[0],[{id:'page',path:'Page.md',format:'md',title:'Page'}]); await loading
  const html=f.$('#documents').innerHTML
  assert.match(html,/class="vault-workspace"/)
  assert.match(html,/data-vault-sidebar/)
  assert.match(html,/No matching vault pages found\./)
  assert.doesNotMatch(html,/data-vault-create/)
})

test('old library failures cannot replace the current loading status', async () => {
  const f=fixture()
  const old=f.controller.load('old','',true,'vault')
  const latest=f.controller.load('latest','',true,'vault')
  f.pending[0].reject(new Error('Old failure'))
  await old
  assert.equal(f.$('#status').textContent,'Opening vault…')
  f.resolve(f.pending[1]); await latest
})

test('abort invalidates responses even when request transport ignores cancellation', async () => {
  const f=fixture()
  const loading=f.controller.load('vault','',true,'vault')
  f.controller.abort()
  f.$('#documents').innerHTML='New surface'
  f.resolve(f.pending[0],[{id:'old',path:'Old.md',title:'Old'}])
  await loading
  assert.equal(f.$('#documents').innerHTML,'New surface')
})

test('switching libraries invalidates pending page content', async () => {
  const f=fixture()
  const loading=f.controller.load('vault','',true,'vault')
  f.resolve(f.pending[0],[{id:'page',path:'Page.md',title:'Page',format:'md',text_url:'/text'}])
  await loading
  f.$('#documents').onclick({target:{closest: selector => selector==='[data-document]' ? {dataset:{document:'page'}} : null}})
  const latest=f.controller.load('other','',true,'vault')
  f.$('#documents .vault-pane').innerHTML='New pane'
  f.resolve(f.pending[1],{content:'Stale page'})
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(f.$('#documents .vault-pane').innerHTML,'New pane')
  f.resolve(f.pending[2]); await latest
})

const tick = () => new Promise(resolve=>setImmediate(resolve))
function clickLink(f, target) {
  let prevented = false
  f.$('#documents').onclick({preventDefault(){prevented=true},target:{closest: selector =>
    selector === 'a[href^="#vault:"]' ? {getAttribute:()=>`#vault:${encodeURIComponent(target)}`} : null}})
  assert.equal(prevented,true)
}

test('controller carries heading into cross-page load and reuses current page for local jumps', async () => {
  const f=fixture(), calls=[]
  const heading={textContent:'Target',setAttribute(){},scrollIntoView(){calls.push('scroll')},focus(){calls.push('focus')}}
  f.$('#documents .vault-pane').querySelectorAll=()=>[heading]
  const loading=f.controller.load('vault','',true,'vault')
  f.resolve(f.pending[0],[
    {id:'start',library_id:'vault',path:'Notes/Start.md',title:'Start',format:'md',text_url:'/start'},
    {id:'target',library_id:'vault',path:'Notes/Target.md',title:'Target',format:'md',text_url:'/target'},
  ])
  await loading
  clickLink(f,'Notes/Start')
  f.resolve(f.pending[1],{content:'Start'}); await tick()
  clickLink(f,'Target#Target')
  assert.equal(f.pending[2].url,'/target')
  assert.deepEqual(calls,[])
  f.resolve(f.pending[2],{content:'Target'}); await tick()
  assert.deepEqual(calls,['scroll','focus'])
  clickLink(f,'#Target')
  clickLink(f,'Target#Target')
  assert.equal(f.pending.length,3)
  assert.deepEqual(calls,['scroll','focus','scroll','focus','scroll','focus'])
})

test('stale heading navigation cannot scroll or focus a newly selected library', async () => {
  const f=fixture(), calls=[]
  f.$('#documents .vault-pane').querySelectorAll=()=>[{
    textContent:'Target',setAttribute(){},scrollIntoView(){calls.push('scroll')},focus(){calls.push('focus')},
  }]
  const loading=f.controller.load('vault','',true,'vault')
  f.resolve(f.pending[0],[{id:'target',library_id:'vault',path:'Target.md',title:'Target',format:'md',text_url:'/target'}])
  await loading
  clickLink(f,'Target#Target')
  const next=f.controller.load('other','',true,'vault')
  f.resolve(f.pending[1],{content:'Target'}); await tick()
  assert.deepEqual(calls,[])
  f.resolve(f.pending[2]); await next
})
