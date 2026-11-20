import assert from 'node:assert/strict'
import test from 'node:test'
import { describeRule, fillRule, readRule } from '../static/smart-collection-form.js'

const fakeRoot = () => {
  const inputs = {}
  return { querySelector: (selector) => (inputs[selector] ??= { value: '', checked: false }), inputs }
}

test('a rule round-trips through the form and keeps only what was filled in', () => {
  const root = fakeRoot()
  fillRule(root, { author: 'Frank Herbert', status: 'unread', added_days: 30 })
  assert.deepEqual(readRule(root), { author: 'Frank Herbert', status: 'unread', added_days: 30 })
  fillRule(root, {})
  assert.deepEqual(readRule(root), {})
})

test('the missing volumes box is part of the rule and is described on the card', () => {
  const root = fakeRoot()
  fillRule(root, { missing_volumes: true })
  assert.deepEqual(readRule(root), { missing_volumes: true })
  assert.equal(describeRule({ missing_volumes: true, author: 'Asato Asato' }), 'Asato Asato · missing volumes')
  fillRule(root, {})
  assert.deepEqual(readRule(root), {})
})

test('whitespace-only fields are left out', () => {
  const root = fakeRoot()
  fillRule(root, {})
  root.inputs['#rule-tag'].value = '   '
  assert.deepEqual(readRule(root), {})
})

test('the card line reads in plain words', () => {
  assert.equal(describeRule({ status: 'unread', series: 'Dune', format: 'epub', added_days: 7 }), 'Not started · Dune · epub · last 7 days')
  assert.equal(describeRule({ query: 'dragons' }), '“dragons”')
  assert.equal(describeRule(undefined), '')
})
