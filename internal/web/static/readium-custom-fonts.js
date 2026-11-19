// Fonts are local to this browser and never sent to the server.
const maxFontBytes = 10 * 1024 * 1024
const maxFonts = 10
const validId = id => /^BoboFont[0-9a-f]{16}$/.test(id)

export function indexedFontStore(indexedDB = globalThis.indexedDB) {
  let opening
  function open() {
    if (!opening) opening = new Promise((resolve, reject) => {
      if (!indexedDB) return reject(new Error('Browser font storage is unavailable'))
      const req = indexedDB.open('mochil-readium-fonts', 1)
      req.onupgradeneeded = () => req.result.createObjectStore('fonts', { keyPath: 'id' })
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error)
      req.onblocked = () => reject(new Error('Browser font storage is blocked'))
    })
    return opening
  }
  async function transaction(mode, action) {
    const db = await open()
    return new Promise((resolve, reject) => {
      const tx = db.transaction('fonts', mode)
      const request = action(tx.objectStore('fonts'))
      tx.oncomplete = () => resolve(request.result)
      tx.onerror = () => reject(tx.error)
      tx.onabort = () => reject(tx.error || new Error('Font storage failed'))
    })
  }
  async function put(font) {
    const db = await open()
    return new Promise((resolve, reject) => {
      const tx = db.transaction('fonts', 'readwrite')
      const store = tx.objectStore('fonts')
      let reason
      const count = store.count()
      count.onsuccess = () => {
        if (count.result >= maxFonts) {
          reason = new Error('Font limit reached (10 fonts).')
          tx.abort()
        } else store.add(font)
      }
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(reason || tx.error || new Error('Font storage failed'))
      tx.onabort = () => reject(reason || tx.error || new Error('Font storage failed'))
    })
  }
  return { list: () => transaction('readonly', s => s.getAll()), put, remove: id => transaction('readwrite', s => s.delete(id)) }
}

export function createCustomFontManager({ store = indexedFontStore(), FontFace = globalThis.FontFace, BroadcastChannel = globalThis.window?.BroadcastChannel, randomId = () => [...crypto.getRandomValues(new Uint8Array(8))].map(b => b.toString(16).padStart(2, '0')).join('') } = {}) {
  let fonts = []
  let pendingAdds = 0
  let revision = 0
  let channel
  try { if (BroadcastChannel) channel = new BroadcastChannel('mochil-readium-fonts') } catch { /* Storage remains usable without notifications. */ }
  const documents = new Map()
  async function installOne(doc, font) {
    let faces = documents.get(doc)
    if (!faces) documents.set(doc, faces = new Map())
    if (faces.has(font.id)) return
    const face = new FontFace(font.id, font.bytes.slice(0))
    await face.load()
    // Removal or a concurrent frame load may have happened while decoding.
    if (documents.get(doc) !== faces || !fonts.some(item => item.id === font.id) || faces.has(font.id)) return
    doc.fonts.add(face)
    faces.set(font.id, face)
  }
  const manager = {
    async load() {
      const current = ++revision
      const saved = await store.list()
      if (current !== revision) return this.list()
      fonts = saved.filter(f => validId(f.id) && typeof f.name === 'string' && f.bytes instanceof ArrayBuffer && f.bytes.byteLength <= maxFontBytes)
      for (const [doc, faces] of documents) {
        for (const [id, face] of faces) if (!fonts.some(font => font.id === id)) { doc.fonts.delete(face); faces.delete(id) }
      }
      await Promise.all([...documents.keys()].flatMap(doc => fonts.map(font => installOne(doc, font))))
      return this.list()
    },
    list() { return fonts.map(({ id, name }) => ({ id, name })) },
    async add(file) {
      if (fonts.length + pendingAdds >= maxFonts) throw new Error('Font limit reached (10 fonts).')
      pendingAdds++
      try {
      if (!file || !/\.(ttf|otf|woff2?)$/i.test(file.name) || !file.size || file.size > maxFontBytes) throw new Error('Choose a TTF, OTF, WOFF or WOFF2 font under 10 MB.')
      const bytes = await file.arrayBuffer()
      if (!bytes.byteLength || bytes.byteLength > maxFontBytes) throw new Error('Invalid font size.')
      const id = 'BoboFont' + randomId()
      if (!validId(id) || fonts.some(f => f.id === id)) throw new Error('Could not create a unique font ID.')
      const font = { id, name: file.name.slice(0, 160), bytes }
      try { await new FontFace(id, bytes.slice(0)).load() } catch { throw new Error('The browser could not read this font.') }
      await store.put(font)
      revision++
      fonts.push(font)
      channel?.postMessage('changed')
      try { await Promise.all([...documents.keys()].map(doc => installOne(doc, font))) }
      catch (error) {
        try { await this.remove(id) } catch { throw new Error('Font was saved, but could not be installed or removed. Reload to manage it.') }
        throw error
      }
      return { id, name: font.name }
      } finally { pendingAdds-- }
    },
    async install(doc) {
      if (!documents.has(doc)) documents.set(doc, new Map())
      await Promise.all(fonts.map(font => installOne(doc, font)))
    },
    async remove(id) {
      if (!fonts.some(f => f.id === id)) return
      await store.remove(id)
      revision++
      fonts = fonts.filter(f => f.id !== id)
      channel?.postMessage('changed')
      for (const [doc, faces] of documents) {
        const face = faces.get(id)
        if (face) { doc.fonts.delete(face); faces.delete(id) }
      }
    },
    forget(doc) { documents.delete(doc) },
    close() { channel?.close() },
  }
  if (channel) channel.onmessage = event => {
    if (event.data !== 'changed') return
    manager.load().then(() => manager.onchange?.(), error => manager.onerror?.(error))
  }
  return manager
}


// The navigator's frameLoaded callback reports its current frame, not necessarily
// the frame which loaded. Observe the actual iframe elements instead.
export function observeFontFrames(host, manager, onError, Observer = MutationObserver) {
  const tracked = new Map()
  function scan() {
    const frames = new Set(host.querySelectorAll('iframe.readium-navigator-iframe'))
    for (const [frame, doc] of tracked) {
      let current
      try { current = frames.has(frame) && frame.contentDocument } catch {}
      if (current !== doc) { manager.forget(doc); tracked.delete(frame) }
    }
    for (const frame of frames) {
      try {
        const doc = frame.contentDocument
        if (doc && !tracked.has(frame)) {
          tracked.set(frame, doc)
          manager.install(doc).catch(onError)
        }
      } catch (error) { onError(error) }
    }
  }
  const observer = new Observer(scan)
  observer.observe(host, { childList: true, subtree: true })
  host.addEventListener('load', scan, true)
  scan()
  return () => {
    observer.disconnect()
    host.removeEventListener('load', scan, true)
    for (const doc of tracked.values()) manager.forget(doc)
    tracked.clear()
  }
}


export async function restoreFontSelection(settings, manager, timeoutMs = 1500) {
  let timer
  try {
    await Promise.race([manager.load(), new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Font storage timed out')), timeoutMs) })])
  } finally { clearTimeout(timer) }
  if (settings.fontFamily.startsWith('BoboFont') && !manager.list().some(font => font.id === settings.fontFamily)) settings.fontFamily = ''
}

export async function removeSelectedFont(settings, manager) {
  const id = settings.fontFamily
  await manager.remove(id)
  if (settings.fontFamily === id) settings.fontFamily = ''
}


export function createFontSelectionActions(settings, manager) {
  let revision = 0
  return {
    select(value) { revision++; settings.fontFamily = value },
    async upload(file) {
      const current = ++revision
      const font = await manager.add(file)
      if (revision === current) settings.fontFamily = font.id
      return font
    },
    invalidate() { revision++ },
  }
}
