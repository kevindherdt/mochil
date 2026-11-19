import test from 'node:test'
import assert from 'node:assert/strict'
import {createVaultMarkdownRenderer} from '../static/vault-markdown.js'

const render = createVaultMarkdownRenderer([
  {id:'image',library_id:'vault',path:'Assets/my_cover_image.png',title:'Cover',format:'png'},
], 'vault')

test('standard Markdown images share safe local resolution and escaped descriptions', () => {
  const html=render('![<cover>](Assets/my_cover_image.png)\n\n![](Assets/my_cover_image.png "Title")','Page.md')
  assert.equal((html.match(/<img /g)||[]).length,2)
  assert.match(html,/alt="&lt;cover&gt;"/)
  assert.match(html,/alt=""/)
})

test('standard image paths decode once and reject malformed or external destinations', () => {
  const local=createVaultMarkdownRenderer([{id:'space',library_id:'vault',path:'A B.png',format:'png'}],'vault')
  assert.match(local('![A](A%20B.png)','Page.md'),/<img /)
  for(const target of ['https://tracker.test/a.png','https%3A%2F%2Ftracker.test/a.png','bad%ZZ.png','Missing/A%20B.png']) {
    assert.doesNotMatch(local(`![image](${target})`,'Page.md'),/<img /)
  }
  assert.doesNotMatch(local('`![A](A%20B.png)`\n\n```\n![A](A%20B.png)\n```','Page.md'),/<img /)
})

test('wiki embeds resolve safely with optional width and escaped alt text', () => {
  const html=render('![[Assets/my_cover_image.png|300]]\n\n![[Assets/my_cover_image.png|<unsafe>]]','Page.md')
  assert.equal((html.match(/<img /g)||[]).length,2)
  assert.match(html,/width="300"/)
  assert.match(html,/alt="&lt;unsafe&gt;"/)
  assert.match(html,/loading="lazy" decoding="async"/)
})

test('fenced and inline code preserve literal embeds and wiki links', () => {
  const html=render('`![[Assets/my_cover_image.png]]`\n\n```\n[[Page]]\n![[Assets/my_cover_image.png]]\n```','Page.md')
  assert.doesNotMatch(html,/<img |<a /)
  assert.match(html,/!\[\[Assets\/my_cover_image.png\]\]/)
})

test('missing and remote embeds remain text links without remote image loading', () => {
  const html=render('![[Missing/my_cover_image.png]] ![[https://tracker.test/a.png]] [[Page|Read more]]','Page.md')
  assert.doesNotMatch(html,/<img /)
  assert.match(html,/Attachment: Missing/)
  assert.match(html,/href="#vault:Page">Read more<\/a>/)
})

test('standard Markdown note and heading links share internal wiki navigation', () => {
  const html=render('[Section](#Caf%C3%A9) [**Next**](../Next.md#Heading) [Other](Other.markdown)','Page.md')
  assert.match(html,/href="#vault:%23Caf%C3%A9">Section/)
  assert.match(html,/href="#vault:..%2FNext.md%23Heading"><strong>Next<\/strong>/)
  assert.match(html,/href="#vault:Other.markdown"/)
})

test('standard external links and code are not converted into vault references', () => {
  const html=render('[External](https://example.test/Note.md#Heading) ` [Note](Note.md) `\n\n```\n[Note](Note.md)\n```','Page.md')
  assert.match(html,/href="https:\/\/example.test\/Note.md#Heading"/)
  assert.doesNotMatch(html,/#vault:/)
  for(const target of ['bad%ZZ.md','javascript:Note.md','//example.test/Note.md','%5C%5Chost/Note.md','Bad%00.md'])
    assert.doesNotMatch(render(`[Link](${target})`,'Page.md'),/#vault:/)
})
