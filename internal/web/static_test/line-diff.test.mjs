import assert from 'node:assert/strict'
import test from 'node:test'
import { diffLines, foldUnchanged, countChanges } from '../static/line-diff.js'

test('diffLines marks added, removed and unchanged lines in order', () => {
  const ops = diffLines('a\nb\nc', 'a\nB\nc\nd')
  assert.deepEqual(ops.map(op => op.type + ':' + op.text), ['same:a', 'del:b', 'add:B', 'same:c', 'add:d'])
  assert.deepEqual(countChanges(ops), { added: 2, removed: 1 })
})

test('identical text has no changes', () => {
  assert.deepEqual(countChanges(diffLines('x\ny', 'x\ny')), { added: 0, removed: 0 })
})

test('long unchanged stretches fold into one row and keep context around edits', () => {
  const before = Array.from({ length: 20 }, (_, i) => 'line' + i).join('\n')
  const after = before.replace('line10', 'LINE10')
  const folded = foldUnchanged(diffLines(before, after), 2)
  assert.deepEqual(folded.filter(op => op.type === 'fold').map(op => op.count), [8, 7])
  assert.equal(folded.filter(op => op.type === 'same').length, 4)
})

test('a note of a thousand lines compares quickly', () => {
  const lines = Array.from({ length: 1000 }, (_, i) => 'row ' + i)
  const started = Date.now()
  diffLines(lines.join('\n'), lines.map((l, i) => (i % 50 ? l : l + '!')).join('\n'))
  assert.ok(Date.now() - started < 1500)
})
