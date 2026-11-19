// Remembers which series or author page is open, so the address can name it and a refresh can open it again.
export function createDetailRestore() {
  let pending = ''
  let opened = ''
  return {
    openOnLoad(name) { pending = name || '' },
    take() { const name = pending; pending = ''; return name },
    set(name) {
      if (opened === (name || '')) return
      opened = name || ''
      globalThis.window?.dispatchEvent(new Event('mochil:place'))
    },
    opened: () => opened,
  }
}
