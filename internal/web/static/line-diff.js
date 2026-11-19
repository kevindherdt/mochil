// A small line-by-line comparison for showing what an edit changed. Notes are a few hundred lines, so the plain
// longest-common-subsequence table is fast enough and easy to trust.
export function diffLines(before, after) {
  const a = before.split('\n'), b = after.split('\n')
  const table = Array.from({ length: a.length + 1 }, () => new Uint16Array(b.length + 1))
  for (let i = a.length - 1; i >= 0; i--)
    for (let j = b.length - 1; j >= 0; j--)
      table[i][j] = a[i] === b[j] ? table[i + 1][j + 1] + 1 : Math.max(table[i + 1][j], table[i][j + 1])
  const ops = []
  let i = 0, j = 0
  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) { ops.push({ type: 'same', text: a[i] }); i++; j++ }
    else if (table[i + 1][j] >= table[i][j + 1]) ops.push({ type: 'del', text: a[i++] })
    else ops.push({ type: 'add', text: b[j++] })
  }
  while (i < a.length) ops.push({ type: 'del', text: a[i++] })
  while (j < b.length) ops.push({ type: 'add', text: b[j++] })
  return ops
}

// Long stretches that did not change are folded into one "N unchanged lines" row, keeping a little context around edits.
export function foldUnchanged(ops, context = 2) {
  const keep = new Array(ops.length).fill(false)
  ops.forEach((op, index) => {
    if (op.type === 'same') return
    for (let k = Math.max(0, index - context); k <= Math.min(ops.length - 1, index + context); k++) keep[k] = true
  })
  const out = []
  for (let index = 0; index < ops.length; index++) {
    if (keep[index]) { out.push(ops[index]); continue }
    let end = index
    while (end < ops.length && !keep[end]) end++
    out.push({ type: 'fold', count: end - index })
    index = end - 1
  }
  return out
}

export function countChanges(ops) {
  return { added: ops.filter(op => op.type === 'add').length, removed: ops.filter(op => op.type === 'del').length }
}
