import assert from 'node:assert/strict'
import test from 'node:test'
import { spatialTargetIndex } from '../static/spatial-navigation.js'

const rect=(left,top,width=100,height=120)=>({left,top,right:left+width,bottom:top+height})

test('spatial navigation follows visual rows and columns',()=>{
  const grid=[rect(0,0),rect(120,0),rect(240,0),rect(0,140),rect(120,140),rect(240,140)]
  assert.equal(spatialTargetIndex(grid,1,'ArrowLeft'),0)
  assert.equal(spatialTargetIndex(grid,1,'ArrowRight'),2)
  assert.equal(spatialTargetIndex(grid,1,'ArrowDown'),4)
  assert.equal(spatialTargetIndex(grid,4,'ArrowUp'),1)
})

test('spatial navigation prefers aligned targets and respects edges',()=>{
  const uneven=[rect(0,0),rect(130,15),rect(0,150),rect(160,180)]
  assert.equal(spatialTargetIndex(uneven,0,'ArrowDown'),2)
  assert.equal(spatialTargetIndex(uneven,0,'ArrowUp'),-1)
  assert.equal(spatialTargetIndex(uneven,3,'ArrowRight'),-1)
})
