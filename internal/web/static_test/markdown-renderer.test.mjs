import assert from 'node:assert/strict'
import test from 'node:test'
import { renderMarkdown } from '../static/markdown-renderer.js'

test('renders structured blocks and joins wrapped paragraphs', () => {
  const html = renderMarkdown(`# Title

First line
continues here.

- One
- Two

1. First
2. Second

> A quoted
> thought

---

\`\`\`js
const unsafe = '<tag>'
\`\`\``)
  assert.match(html, /<h1>Title<\/h1>/)
  assert.match(html, /<p>First line continues here\.<\/p>/)
  assert.match(html, /<ul><li>One<\/li><li>Two<\/li><\/ul>/)
  assert.match(html, /<ol><li>First<\/li><li>Second<\/li><\/ol>/)
  assert.match(html, /<blockquote><p>A quoted thought<\/p><\/blockquote>/)
  assert.match(html, /<hr>/)
  assert.match(html, /const unsafe = &#39;&lt;tag&gt;&#39;/)
})

test('supports safe inline formatting and links', () => {
  const html = renderMarkdown('Use **strong**, *emphasis*, `code`, and [docs](https://example.com/a?b=1&c=2).')
  assert.match(html, /<strong>strong<\/strong>/)
  assert.match(html, /<em>emphasis<\/em>/)
  assert.match(html, /<code>code<\/code>/)
  assert.match(html, /href="https:\/\/example\.com\/a\?b=1&amp;c=2"/)
})

test('escapes raw HTML and refuses active or protocol-relative links', () => {
  const html = renderMarkdown('<img src=x onerror=alert(1)> [bad](javascript:alert(1)) [also bad](//evil.example)')
  assert.doesNotMatch(html, /<img|href=/)
  assert.match(html, /&lt;img src=x onerror=alert\(1\)&gt;/)
  assert.match(html, /\[bad\]\(javascript:alert\(1\)\)/)
})
