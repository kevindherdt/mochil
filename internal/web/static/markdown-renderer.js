import { createMarkdownLineCursor } from './markdown-lines.js?v=1'
import { markdownTableCells, markdownTableHeader } from './markdown-table.js?v=1'

const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, character => ({
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
})[character])

function safeLink(destination) {
  const value = destination.trim()
  // Browsers normalize backslashes in special URLs; /\\host is not a local path.
  if (/[\\\u0000-\u001f\u007f]/.test(value)) return ''
  if (/^(?:https?:\/\/|mailto:|\/(?!\/)|\.\.?\/|#)/i.test(value)) return value
  return ''
}

function inline(value, depth = 0, options = {}) {
  if (depth > 3) return escapeHTML(value)
  const token = /(`[^`\n]*`)|\[([^\]\n]+)\]\(([^\s)]+)(?:\s+"[^"]*")?\)|(\*\*|__)(.+?)\4|(\*|_)([^\n]+?)\6|(!?\[\[[^\]\n]+\]\])|(!\[([^\]\n]*)\]\(([^\s)]+)(?:\s+"[^"]*")?\))/g
  let output = '', offset = 0, match
  while ((match = token.exec(value))) {
    output += (options.renderText || escapeHTML)(value.slice(offset, match.index))
    if (match[1]) {
      output += `<code>${escapeHTML(match[1].slice(1, -1))}</code>`
    } else if (match[2] !== undefined) {
      const label = inline(match[2], depth + 1, options)
      const custom = options.renderLink?.(match[3], label)
      if (custom != null) {
        output += custom
        offset = token.lastIndex
        continue
      }
      const href = safeLink(match[3])
      output += href
        ? `<a href="${escapeHTML(href)}" rel="noopener noreferrer">${label}</a>`
        : escapeHTML(match[0])
    } else if (match[9]) {
      output += options.renderImage?.(match[11], match[10]) ?? escapeHTML(match[9])
    } else if (match[8]) {
      output += (options.renderText || escapeHTML)(match[8])
    } else if (match[4]) {
      output += `<strong>${inline(match[5], depth + 1, options)}</strong>`
    } else {
      output += `<em>${inline(match[7], depth + 1, options)}</em>`
    }
    offset = token.lastIndex
  }
  return output + (options.renderText || escapeHTML)(value.slice(offset))
}

export function renderMarkdown(value, options = {}) {
  const lines = createMarkdownLineCursor(value)
  const output = []

  while (lines.peek() !== null) {
    const lineOffset = lines.position()
    const line = lines.take()
    if (!line.trim()) continue

    const fence = line.match(/^\s*```/)
    if (fence) {
      const code = []
      while (lines.peek() !== null && !/^\s*```\s*$/.test(lines.peek())) code.push(lines.take())
      if (lines.peek() !== null) lines.take()
      output.push(`<pre><code>${escapeHTML(code.join('\n'))}</code></pre>`)
      continue
    }

    const heading = line.match(/^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$/)
    if (heading) {
      const level = heading[1].length
      output.push(`<h${level}>${inline(heading[2], 0, options)}</h${level}>`)
      continue
    }

    const table = options.tables && markdownTableHeader(line, lines.peek())
    if (table) {
      lines.take()
      const cells = (values, tag) => values.map((value, index) => `<${tag}${tag === 'th' ? ' scope="col"' : ''} style="text-align:${table.alignments[index]}">${inline(value, 0, options)}</${tag}>`).join('')
      const rows = []
      while (lines.peek() !== null) {
        const values = markdownTableCells(lines.peek())
        if (!values || values.length !== table.headers.length) break
        lines.take()
        rows.push(`<tr>${cells(values, 'td')}</tr>`)
      }
      output.push(`<div class="vault-table-scroll" tabindex="0" role="region" aria-label="Table"><table><thead><tr>${cells(table.headers, 'th')}</tr></thead><tbody>${rows.join('')}</tbody></table></div>`)
      continue
    }

    if (/^\s{0,3}(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
      output.push('<hr>')
      continue
    }

    const list = line.match(/^\s{0,3}([-+*]|\d+[.)])\s+(.+)/)
    if (list) {
      const ordered = /^\d/.test(list[1])
      const tag = ordered ? 'ol' : 'ul'
      const listItem = (match, offset) => (!ordered && options.renderTask?.(match[2], offset + match[0].length - match[2].length, text => inline(text, 0, options))) || `<li>${inline(match[2], 0, options)}</li>`
      const items = [listItem(list, lineOffset)]
      while (lines.peek() !== null) {
        const item = lines.peek().match(/^\s{0,3}([-+*]|\d+[.)])\s+(.+)/)
        if (!item || /^\d/.test(item[1]) !== ordered) break
        items.push(listItem(item, lines.position()))
        lines.take()
      }
      output.push(`<${tag}>${items.join('')}</${tag}>`)
      continue
    }

    if (/^\s{0,3}>/.test(line)) {
      const quote = [line.match(/^\s{0,3}>\s?(.*)$/)[1]]
      while (lines.peek() !== null) {
        const part = lines.peek().match(/^\s{0,3}>\s?(.*)$/)
        if (!part) break
        quote.push(part[1])
        lines.take()
      }
      output.push(`<blockquote><p>${inline(quote.join(' '), 0, options)}</p></blockquote>`)
      continue
    }

    const paragraph = [line.trim()]
    while (lines.peek() !== null && lines.peek().trim() &&
      !/^\s{0,3}(?:#{1,6}\s+|```|>|[-+*]\s+|\d+[.)]\s+|(?:-{3,}|\*{3,}|_{3,})\s*$)/.test(lines.peek())) {
      paragraph.push(lines.take().trim())
    }
    output.push(`<p>${inline(paragraph.join(' '), 0, options)}</p>`)
  }

  return output.join('\n')
}
