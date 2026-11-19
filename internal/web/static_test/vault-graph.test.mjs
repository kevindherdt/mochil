import test from 'node:test'
import assert from 'node:assert/strict'
import { selectGraph, layoutGraph } from '../static/vault-graph-layout.js'

const graph = {
  nodes: ['a', 'b', 'c', 'd', 'e', 'f'].map(id => ({ id, title: id.toUpperCase(), path: `${id}.md` })),
  edges: [[0, 1], [1, 2], [2, 3], [0, 2]], // e and f are not linked to anything
}

test('"all" keeps the best-connected pages first and counts the rest as hidden', () => {
  const all = selectGraph(graph, { mode: 'all', cap: 3 })
  assert.deepEqual(all.nodes.map(n => n.id), ['a', 'b', 'c'])
  assert.equal(all.hidden, 3)
  assert.equal(all.edges.length, 3)
  assert.equal(all.nodes.find(n => n.id === 'c').degree, 3)
  assert.equal(selectGraph(graph, { mode: 'all', cap: 10 }).nodes.length, 6, 'unlinked pages are shown when there is room')
})

test('"local" is the open page and everything within two links', () => {
  const local = selectGraph(graph, { mode: 'local', activeId: 'a' })
  assert.deepEqual(local.nodes.map(n => n.id), ['a', 'b', 'c', 'd'])
  const lone = selectGraph(graph, { mode: 'local', activeId: 'e' })
  assert.deepEqual(lone.nodes.map(n => n.id), ['e'])
  assert.deepEqual(selectGraph(graph, { mode: 'local', activeId: 'zzz' }).nodes.length, 6, 'unknown page falls back to the whole graph')
})

test('layout stays in the unit square, is repeatable, and pulls linked pages together', () => {
  const picked = selectGraph(graph, { mode: 'all', cap: 10 })
  const one = layoutGraph(picked.nodes, picked.edges), two = layoutGraph(picked.nodes, picked.edges)
  assert.deepEqual([...one.x], [...two.x])
  for (let i = 0; i < picked.nodes.length; i++) {
    assert.ok(one.x[i] >= 0 && one.x[i] <= 1 && one.y[i] >= 0 && one.y[i] <= 1 && Number.isFinite(one.x[i]))
  }
  const dist = (a, b) => Math.hypot(one.x[a] - one.x[b], one.y[a] - one.y[b])
  assert.ok(dist(0, 1) < dist(0, 3), 'a and b are linked, a and d are two steps apart')
})

test('empty and single-page graphs do not break the layout', () => {
  assert.equal(layoutGraph([], []).x.length, 0)
  const single = layoutGraph([{ id: 'x' }], [])
  assert.ok(Number.isFinite(single.x[0]))
})
