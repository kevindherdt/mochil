import assert from 'node:assert/strict'
import test from 'node:test'
import { createSerializedWriter } from '../static/serialized-writer.js'

const tick=()=>new Promise(resolve=>setTimeout(resolve,0))

test('serializes writes and coalesces pending values to the newest state',async()=>{const sent=[],release=[];const write=createSerializedWriter(value=>new Promise(resolve=>{sent.push(value);release.push(resolve)}));write('first');write('second');write('third');await tick();assert.deepEqual(sent,['first']);release.shift()();await tick();assert.deepEqual(sent,['first','third']);release.shift()();await tick()})

test('continues after a failed write and reports it',async()=>{const sent=[],errors=[];const write=createSerializedWriter(async value=>{sent.push(value);if(value==='first')throw new Error('failed')},error=>errors.push(error.message));write('first');await tick();write('second');await tick();assert.deepEqual(sent,['first','second']);assert.deepEqual(errors,['failed'])})
