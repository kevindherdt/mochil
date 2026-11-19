import test from 'node:test'
import assert from 'node:assert/strict'
import {createMarkdownLineCursor} from '../static/markdown-lines.js'
import {renderMarkdown} from '../static/markdown-renderer.js'

test('line cursor matches normalized split for mixed endings and trailing empty lines',()=>{
  const fragments=['','text','雪😀','\n','\r','\r\n','text\r\nnext','\n\n']
  for(const first of fragments)for(const second of fragments)for(const third of fragments){
    const text=first+second+third,cursor=createMarkdownLineCursor(text),lines=[]
    while(cursor.peek()!==null){assert.equal(cursor.peek(),cursor.peek());lines.push(cursor.take())}
    assert.deepEqual(lines,text.replace(/\r\n?/g,'\n').split('\n'))
    assert.equal(cursor.take(),null)
    assert.equal(cursor.peek(),null)
    assert.equal(cursor.position(),text.length)
  }
})

test('sequential renderer retains adjacent block boundaries with mixed line endings',()=>{
  const source='# Heading\r\n\r\n- one\r- two\n1. ordered\r\n> quoted\r> again\n\n```js\r\nconst x = 1\r\n```\nparagraph\rcontinued\n---'
  assert.equal(renderMarkdown(source),[
    '<h1>Heading</h1>','<ul><li>one</li><li>two</li></ul>','<ol><li>ordered</li></ol>',
    '<blockquote><p>quoted again</p></blockquote>','<pre><code>const x = 1</code></pre>',
    '<p>paragraph continued</p>','<hr>',
  ].join('\n'))
})

test('unterminated fences preserve final empty lines and HTML escaping',()=>{
  assert.equal(renderMarkdown('```\r\n<a>\r\n'),'<pre><code>&lt;a&gt;\n</code></pre>')
  assert.equal(renderMarkdown(null),'')
})

test('line cursor exposes original character offsets without normalizing the source',()=>{
  for(const newline of ['\n','\r\n','\r']) {
    const chunks=['\uFEFF---','title: 雪😀','---','- [ ] repeat','```','- [ ] repeat','```','- [ ] repeat']
    const source=chunks.join(newline), cursor=createMarkdownLineCursor(source)
    let position=0
    for(const chunk of chunks) {
      assert.equal(cursor.position(),position)
      assert.equal(source.slice(cursor.position(),cursor.position()+chunk.length),chunk)
      assert.equal(cursor.peek(),chunk)
      assert.equal(cursor.position(),position)
      assert.equal(cursor.take(),chunk)
      position+=chunk.length+newline.length
    }
    assert.equal(cursor.peek(),null)
    assert.equal(cursor.position(),source.length)
  }
})
