// The wiki's note editor: Markdown stays the file, but it reads like the page
// while you write. Saves on its own a moment after you stop typing. Loaded only
// when someone edits; if it cannot load, the plain-text editor takes over.
import {
  EditorState, EditorView, Decoration, ViewPlugin, keymap, history, historyKeymap, defaultKeymap,
  indentWithTab, markdown, markdownLanguage, syntaxTree, autocompletion, WidgetType,
} from '/assets/vendor/codemirror/codemirror.js'
import { previewSpecs } from './vault-live-preview.js?v=4'
import { formatTools, toolbarHTML } from './vault-live-commands.js?v=3'
import { wikiLinkSource } from './vault-link-complete.js?v=1'
import { vaultMutationCommitted } from './vault-mutation-error.js?v=1'

const AUTOSAVE_MS = 1100
const RETRY_MS = 4000
const margin = 2000

// A picture shown in place of its ![[embed]] text.
class ImageWidget extends WidgetType {
  constructor(url, alt, width) { super(); this.url = url; this.alt = alt; this.width = width }
  eq(other) { return other.url === this.url && other.width === this.width }
  toDOM() {
    const image = document.createElement('img')
    image.className = 'cm-md-img'
    image.src = this.url
    image.alt = this.alt
    image.loading = 'lazy'
    if (this.width) image.style.width = `${Math.min(this.width, 4096)}px`
    return image
  }
}

// Set by mountLiveEditor: finds the address of an image by the name written in the note.
let imageAddress = () => null

function decorations(view) {
  const { state } = view
  const first = view.visibleRanges[0], last = view.visibleRanges.at(-1)
  if (!first) return Decoration.none
  const visible = { from: Math.max(0, first.from - margin), to: Math.min(state.doc.length, last.to + margin) }
  const ranges = state.selection.ranges.map(range => ({ from: range.from, to: range.to }))
  const built = []
  for (const spec of previewSpecs({ tree: syntaxTree(state), doc: state.doc, ranges, visible, isImage: target => imageAddress(target) !== null })) {
    if (spec.kind === 'hide') {
      // Hidden text never spans a line break.
      if (!state.doc.sliceString(spec.from, spec.to).includes('\n')) built.push(Decoration.replace({}).range(spec.from, spec.to))
    } else if (spec.kind === 'image') {
      built.push(Decoration.replace({ widget: new ImageWidget(imageAddress(spec.target), spec.target, spec.width) }).range(spec.from, spec.to))
    } else if (spec.kind === 'mark') built.push(Decoration.mark({ class: spec.className }).range(spec.from, spec.to))
    else built.push(Decoration.line({ class: spec.className }).range(spec.from))
  }
  return Decoration.set(built, true)
}

const livePreview = ViewPlugin.fromClass(class {
  constructor(view) { this.decorations = decorations(view) }
  update(update) {
    if (update.docChanged || update.selectionSet || update.viewportChanged || update.focusChanged) this.decorations = decorations(update.view)
  }
}, { decorations: plugin => plugin.decorations })

export function mountLiveEditor({ pane, item, content, revision, libraryName, escapeHTML, requestJSON, onClose, pages = [], resolveImage = () => null, pickTemplate = null }) {
  const folders = String(item.path || '').split('/').filter(Boolean).slice(0, -1)
  const crumbs = [libraryName || 'Vault', ...folders].map(part => `<span>${escapeHTML(part)}</span>`).join('<i aria-hidden="true">/</i>')
  pane.classList.add('vault-live-pane')
  pane.innerHTML = `<header class="vault-notebar"><div class="vault-notebar-column"><nav class="vault-crumbs" aria-label="Location">${crumbs}</nav><div class="vault-format" role="toolbar" aria-label="Formatting">${toolbarHTML(formatTools.filter(tool => tool.action !== 'template' || pickTemplate?.available?.()))}</div><div class="vault-page-actions"><span class="vault-edit-state" role="status" aria-live="polite">Saved</span><button type="button" data-vault-done class="vault-done">Done</button></div></div></header>
    <h1 class="vault-title">${escapeHTML(item.title)}</h1>
    <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple hidden data-vault-image-input>
    <div class="vault-live-host"></div>`
  const host = pane.querySelector('.vault-live-host')
  const state = pane.querySelector('.vault-edit-state')
  const done = pane.querySelector('[data-vault-done]')

  let current = revision
  let dirty = false
  let saving = null
  let halted = false
  let timer = 0
  let closed = false
  const url = `/api/vaults/${encodeURIComponent(item.library_id)}/pages/${encodeURIComponent(item.id)}`
  const say = (text, kind = '') => { state.textContent = text; state.dataset.kind = kind }

  async function save() {
    clearTimeout(timer)
    if (saving) return saving
    if (!dirty || halted || closed) return
    saving = (async () => {
      while (dirty && !halted && !closed) {
        const text = view.state.doc.toString()
        dirty = false
        say('Saving…')
        try {
          const result = await requestJSON(url, {
            method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ revision: current, content: text }),
          })
          if (!result.data?.revision) { halted = true; say('Saved. Close and reopen the page to keep editing.', 'warn'); break }
          current = result.data.revision
          if (!dirty) say('Saved')
        } catch (error) {
          dirty = true
          if (error.status === 409) {
            halted = true
            say('This page changed somewhere else.', 'error')
            showConflict(text)
          } else if (vaultMutationCommitted(error)) {
            halted = true; dirty = false
            say('Saved. Refresh the library before editing again.', 'warn')
          } else {
            say('Not saved. Retrying…', 'error')
            timer = setTimeout(save, RETRY_MS)
          }
          break
        }
      }
    })().finally(() => { saving = null })
    return saving
  }
  // Pictures are stored in the vault; the note keeps an ![[embed]] of each.
  const uploaded = new Map()
  imageAddress = target => uploaded.get(target.toLowerCase()) ?? resolveImage(target)
  const imageFiles = transfer => [...(transfer?.files || [])].filter(file => /^image\/(png|jpeg|gif|webp)$/.test(file.type))
  async function addImages(files) {
    for (const file of files) {
      const generic = !file.name || /^image\.(png|jpe?g|gif|webp)$/i.test(file.name)
      const stamp = new Date().toISOString().replace(/\D/g, '').slice(0, 14)
      const name = generic ? `Pasted image ${stamp}.${file.type.split('/')[1].replace('jpeg', 'jpg')}` : file.name
      say('Adding image…')
      try {
        const { data } = await requestJSON(`/api/vaults/${encodeURIComponent(item.library_id)}/attachments?name=${encodeURIComponent(name)}`, {
          method: 'POST', headers: { 'Content-Type': file.type }, body: file,
        })
        if (data.url) uploaded.set(data.name.toLowerCase(), data.url)
        const { from, to } = view.state.selection.main
        const own = from > 0 && view.state.sliceDoc(from - 1, from) !== '\n' ? '\n' : ''
        const embed = `${own}![[${data.name}]]\n`
        view.dispatch({ changes: { from, to, insert: embed }, selection: { anchor: from + embed.length }, userEvent: 'input.paste' })
      } catch (error) {
        say(error.message || 'Could not add that image.', 'error')
        return
      }
    }
  }
  const queue = () => { dirty = true; say('Editing…'); clearTimeout(timer); timer = setTimeout(save, AUTOSAVE_MS) }

  function showConflict(mine) {
    done.textContent = 'Reload page'
    const copy = document.createElement('button')
    copy.type = 'button'
    copy.textContent = 'Copy my text'
    copy.onclick = async () => { try { await navigator.clipboard.writeText(mine); copy.textContent = 'Copied' } catch { copy.textContent = 'Select the text and copy it' } }
    state.after(copy)
    view.contentDOM.contentEditable = 'false'
  }

  const keys = formatTools.filter(tool => tool.key).map(tool => ({
    key: tool.key, preventDefault: true,
    run: target => { target.dispatch(target.state.update(tool.run(target.state), { scrollIntoView: true, userEvent: 'input' })); return true },
  }))
  const view = new EditorView({
    parent: host,
    state: EditorState.create({
      doc: content,
      extensions: [
        history(), markdown({ base: markdownLanguage }), livePreview,
        autocompletion({ override: [wikiLinkSource(() => pages)], icons: false }), EditorView.lineWrapping,
        keymap.of([...keys, { key: 'Mod-s', preventDefault: true, run: () => { save(); return true } }, indentWithTab, ...historyKeymap, ...defaultKeymap]),
        EditorView.contentAttributes.of({ 'aria-label': `Edit ${item.title}`, spellcheck: 'true', autocapitalize: 'sentences' }),
        EditorView.updateListener.of(update => { if (update.docChanged && !halted) queue() }),
        EditorView.domEventHandlers({
          blur: () => { if (dirty) save() },
          paste: event => { const files = imageFiles(event.clipboardData); if (!files.length || halted) return false; event.preventDefault(); addImages(files); return true },
          drop: event => { const files = imageFiles(event.dataTransfer); if (!files.length || halted) return false; event.preventDefault(); addImages(files); return true },
        }),
      ],
    }),
  })
  view.focus()

  pane.querySelector('.vault-format').addEventListener('pointerdown', event => event.preventDefault())
  pane.querySelector('.vault-format').addEventListener('click', event => {
    const tool = formatTools.find(candidate => candidate.id === event.target.closest('[data-format]')?.dataset.format)
    if (!tool || halted) return
    if (tool.action === 'template') {
      pickTemplate(text => {
        const { from, to } = view.state.selection.main
        // A template is a block of its own: start it on a fresh line.
        const insert = (from > 0 && view.state.sliceDoc(from - 1, from) !== '\n' ? '\n' : '') + text
        view.dispatch({ changes: { from, to, insert }, selection: { anchor: from + insert.length }, scrollIntoView: true, userEvent: 'input.paste' })
        view.focus()
      })
      return
    }
    if (tool.action === 'image') { pane.querySelector('[data-vault-image-input]').click(); return }
    view.dispatch(view.state.update(tool.run(view.state), { scrollIntoView: true, userEvent: 'input' }))
    view.focus()
  })

  const picker = pane.querySelector('[data-vault-image-input]')
  picker.addEventListener('change', () => { addImages(imageFiles({ files: picker.files })); picker.value = '' })

  const leaving = event => { if (dirty || saving) { event.preventDefault(); event.returnValue = '' } }
  const hidden = () => { if (document.visibilityState === 'hidden') save() }
  window.addEventListener('beforeunload', leaving)
  document.addEventListener('visibilitychange', hidden)
  const teardown = () => {
    closed = true
    clearTimeout(timer)
    window.removeEventListener('beforeunload', leaving)
    document.removeEventListener('visibilitychange', hidden)
    view.destroy()
    pane.classList.remove('vault-live-pane')
  }
  done.onclick = async () => {
    done.disabled = true
    if (!halted) { await save(); if (dirty) { done.disabled = false; return } }
    teardown()
    await onClose()
  }
  return { destroy: teardown, flush: save }
}
