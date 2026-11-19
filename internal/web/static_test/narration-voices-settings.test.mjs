import test from 'node:test'
import assert from 'node:assert/strict'
import { describeVoice, voiceCardsHTML, languageChoices, filterVoices, chipsHTML } from '../static/narration-voices-settings.js'

test('a voice is described by its language and length', () => {
  assert.equal(describeVoice({ language: 'dutch', seconds: 17.97 }), 'Dutch · 18 s')
  assert.equal(describeVoice({ language: 'english' }), 'English')
  assert.equal(describeVoice({}), 'English')
})

const voices = [
  { id: 'a', name: 'Alba', language: 'english' },
  { id: 'b', name: 'Cosette', language: 'english' },
  { id: 'c', name: 'Femke', language: 'dutch' },
  { id: 'd', name: 'My Voice', language: 'english', custom: true, seconds: 12 },
]

test('voice names are escaped before they reach the page', () => {
  const html = voiceCardsHTML([{ id: 'a'.repeat(32), name: '<img src=x onerror=alert(1)>', language: 'english', seconds: 12, custom: true }])
  assert.ok(!html.includes('<img'), html)
  assert.match(html, /&lt;img/)
  for (const action of ['listen', 'try', 'rename', 'delete']) assert.ok(html.includes(`data-action="${action}"`))
})

test('built-in voices can be heard but not renamed or deleted', () => {
  const html = voiceCardsHTML([voices[0]])
  assert.ok(html.includes('data-action="try"'))
  for (const action of ['listen', 'rename', 'delete']) assert.ok(!html.includes(`data-action="${action}"`), action)
})

test('the language buttons come from the voices that really exist, with counts', () => {
  assert.deepEqual(languageChoices(voices).map(choice => `${choice.id}:${choice.count}`), ['all:4', 'dutch:1', 'english:3', 'mine:1'])
  assert.deepEqual(languageChoices([{ language: 'english' }]).map(choice => choice.id), ['all', 'english'])
  assert.deepEqual(languageChoices([]).map(choice => choice.id), ['all'])
  assert.ok(!languageChoices([{ language: 'english', custom: true }]).some(choice => choice.id === 'mine'), '"My voices" would just repeat "All"')
})

test('a new language appears without any code change', () => {
  assert.deepEqual(languageChoices([{ language: 'french' }]).map(choice => choice.label), ['All', 'French'])
})

test('voices are filtered by language, by "mine", and by what is typed', () => {
  const names = filter => filterVoices(voices, filter).map(voice => voice.name)
  assert.deepEqual(names({}), ['Alba', 'Cosette', 'Femke', 'My Voice'])
  assert.deepEqual(names({ language: 'dutch' }), ['Femke'])
  assert.deepEqual(names({ language: 'mine' }), ['My Voice'])
  assert.deepEqual(names({ query: ' cos ' }), ['Cosette'])
  assert.deepEqual(names({ language: 'english', query: 'my' }), ['My Voice'])
  assert.deepEqual(names({ query: 'dutch' }), ['Femke'], 'the language word also finds voices')
  assert.deepEqual(names({ query: 'zzz' }), [])
})

test('the selected language button is marked for assistive technology', () => {
  const html = chipsHTML(languageChoices(voices), 'dutch')
  assert.match(html, /data-language="dutch"[^>]*aria-pressed="true"/)
  assert.match(html, /data-language="all"[^>]*aria-pressed="false"/)
})

test('an ordinary account sees no rename or delete for a voice', () => {
  const voice = { id: 'a'.repeat(32), name: 'Mine', language: 'english', seconds: 5, custom: true }
  const admin = voiceCardsHTML([voice], true), reader = voiceCardsHTML([voice], false)
  assert.match(admin, /data-action="rename"/)
  assert.match(admin, /data-action="delete"/)
  assert.doesNotMatch(reader, /data-action="rename"|data-action="delete"/)
  assert.match(reader, /data-action="listen"/)
})
