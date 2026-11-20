import {renderMarkdown} from './markdown-renderer.js?v=1'
import {createVaultAssetResolver} from './vault-assets.js?v=1'
import {splitVaultFrontmatter, parseProperties} from './vault-frontmatter.js?v=2'
import {propertyHref} from './vault-properties.js?v=3'

const escapeHTML = value => String(value).replace(/[&<>"']/g, character => ({
  '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#39;',
})[character])

function imageHTML(image, alt, width = 0) {
  return `<img class="vault-embed-image" src="${escapeHTML(image.url)}" alt="${escapeHTML(alt)}" loading="lazy" decoding="async"${width > 0 ? ` width="${Math.min(width, 4096)}"` : ''}>`
}

const tagPill = tag => `<a class="vault-tag" href="#tag:${encodeURIComponent(tag.toLowerCase())}">#${escapeHTML(tag)}</a>`

function propertyValue(key, value) {
  const wiki = /^\[\[([^\]|]+)(?:\|([^\]]+))?\]\]$/.exec(value)
  if (wiki) return `<a href="#vault:${encodeURIComponent(wiki[1].trim())}">${escapeHTML((wiki[2] || wiki[1]).trim())}</a>`
  if (/^tags?$/i.test(key)) return tagPill(value.replace(/^#/, ''))
  if (value.length <= 60) return `<a class="vault-prop" href="${propertyHref(key, value)}">${escapeHTML(value)}</a>`
  return escapeHTML(value)
}

function propertiesHTML(text, writable = false) {
  const rows = parseProperties(text)
  if (!rows) return `<details class="vault-properties"><summary>Properties</summary><pre><code>${escapeHTML(text)}</code></pre></details>`
  const body = rows.map(row => `<div><dt>${escapeHTML(row.key)}</dt><dd>${row.values.map(value => `<span${row.list ? ' class="vault-chip"' : ''}>${propertyValue(row.key, value)}</span>`).join('')}</dd></div>`).join('')
  return `<details class="vault-properties"><summary>Properties</summary><dl>${body}</dl>${writable ? '<button type="button" class="vault-properties-edit" data-edit-properties>Edit</button>' : ''}</details>`
}

// A #tag starts a word and is not digits only; headings ("# Title") never match.
const inlineTag = /(^|[\s(])#([\p{L}\p{N}_/-]*[\p{L}_/-][\p{L}\p{N}_/-]*)/gu
function withTags(html) {
  return html.replace(inlineTag, (_all, lead, tag) => `${lead}${tagPill(tag)}`)
}

/** Preserve code examples; interpret wiki references only in normal inline text. */
export function createVaultMarkdownRenderer(entries, libraryId) {
  const resolve = createVaultAssetResolver(entries, libraryId)
  return (value, pagePath, {writable = false} = {}) => {
    const {properties, body} = splitVaultFrontmatter(value)
    const metadata = properties?.trim() ? propertiesHTML(properties, writable) : ''
    return metadata + renderMarkdown(body, {
    tables: true,
    renderTask(text, offset, renderInline) {
      const task = /^\[([ xX])\]\s+(.*)$/.exec(text)
      if (!task) return null
      const marker = String(value ?? '').length - body.length + offset + 1
      return `<li class="vault-task"><label><input type="checkbox" data-vault-task="${marker}"${task[1] !== ' ' ? ' checked' : ''}${writable ? '' : ' disabled'}><span>${renderInline(task[2])}</span></label></li>`
    },
    renderLink(destination, label) {
      let target
      try { target = decodeURIComponent(destination) } catch { return null }
      if (!target || target.startsWith('/') || /^[a-z][a-z0-9+.-]*:/i.test(target) || /[\\\u0000-\u001f\u007f]/.test(target)) return null
      const path = target.split('#')[0]
      if (path && !/\.(md|markdown)$/i.test(path)) return null
      return `<a href="#vault:${encodeURIComponent(target)}">${label}</a>`
    },
    renderImage(target, alt) {
      // Markdown destinations are URL-encoded; wiki embed paths are literal.
      let decoded
      try { decoded = decodeURIComponent(target) } catch { return null }
      const image = resolve(pagePath, decoded)
      return image ? imageHTML(image, alt) : null
    },
    renderText(text) {
    const token = /(!?)\[\[([^\]|\n]+)(?:\|([^\]\n]+))?\]\]/g
    let result = '', offset = 0
    for (const match of text.matchAll(token)) {
      result += withTags(escapeHTML(text.slice(offset, match.index)))
      const target = match[2].trim(), label = (match[3] || target).trim()
      const image = match[1] ? resolve(pagePath, target) : null
      if (image) {
        const width = /^\d{1,4}$/.test(match[3] || '') ? Number(match[3]) : 0
        const alt = width ? image.entry.title || target : label
        result += imageHTML(image, alt, width)
      } else {
        result += `<a href="#vault:${encodeURIComponent(target)}">${escapeHTML((match[1] ? 'Attachment: ' : '') + label)}</a>`
      }
      offset = match.index + match[0].length
    }
    return result + withTags(escapeHTML(text.slice(offset)))
  }})
  }
}
