import assert from 'node:assert/strict'

// Only the disposable release server with public synthetic fixture credentials.
const origin = new URL(process.argv[2])
assert.equal(origin.hostname,'127.0.0.1')
assert.equal(origin.protocol,'http:')
assert(origin.port && origin.port !== '8099')
assert.equal(origin.pathname,'/')
async function request(path,options={}) {
  return fetch(new URL(path,origin),{...options,redirect:'error',signal:AbortSignal.timeout(30_000)})
}
const login=await request('/api/android/v1/login',{
  method:'POST',headers:{Authorization:`Basic ${Buffer.from('release-smoke:temporary-release-fixture-only').toString('base64')}`,'Content-Type':'application/json'},
  body:JSON.stringify({device_name:'Playlist release fixture'}),
})
assert.equal(login.status,201)
const {device_token}=await login.json()
assert.equal(typeof device_token,'string')
const headers={Authorization:`Bearer ${device_token}`,'Content-Type':'application/json'}
const collection='/api/android/v1/audio-playlists'
const name=`Fixture-${crypto.randomUUID()}`
const path=`${collection}/${encodeURIComponent(name)}`
let created=false
try {
  const create=await request(collection,{method:'POST',headers,body:JSON.stringify({name})})
  assert.equal(create.status,201)
  created=true
  assert.equal((await request(collection,{method:'POST',headers,body:JSON.stringify({name})})).status,409)
  const original=await request(path,{headers})
  assert.equal(original.status,200)
  const revision=original.headers.get('ETag')
  assert(revision)
  // Case-only rename changes stored state while retaining case-insensitive identity.
  assert.equal((await request(path,{method:'PATCH',headers:{...headers,'If-Match':revision},body:JSON.stringify({name:name.toUpperCase()})})).status,204)
  assert.equal((await request(path,{method:'PUT',headers:{...headers,'If-Match':revision},body:'[]'})).status,412)
  assert.equal((await request(path,{method:'PATCH',headers:{...headers,'If-Match':revision},body:JSON.stringify({name:'Rejected rename'})})).status,412)
  assert.equal((await request(path,{method:'DELETE',headers:{...headers,'If-Match':revision}})).status,412)
  const latest=await request(path,{headers})
  assert.equal(latest.status,200)
  const current=latest.headers.get('ETag')
  assert(current && current!==revision)
  const saved=await request(path,{method:'PUT',headers:{...headers,'If-Match':current},body:'[]'})
  assert.equal(saved.status,204)
  assert(saved.headers.get('ETag'))
  assert.equal((await request(path,{method:'DELETE',headers:{...headers,'If-Match':saved.headers.get('ETag')}})).status,204)
  created=false
  assert.equal((await request(path,{headers})).status,404)
  console.log('Authenticated playlist duplicate-create, conditional replacement, rename and delete checks passed')
} finally {
  if(created)assert.equal((await request(path,{method:'DELETE',headers})).status,204)
}
