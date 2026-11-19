export function createBoundedWindow({ container, topSpacer, bottomSpacer, maximum = 600, trim = 120, onRemove = () => {} }) {
  const height = element => Number.parseFloat(element.style.height) || 0
  const setHeight = (element, value) => { element.style.height = `${Math.max(0, value)}px` }

  function reset() {
    setHeight(topSpacer, 0)
    setHeight(bottomSpacer, 0)
  }

  function begin(mode) {
    const anchor = mode === 'prepend' ? container.firstElementChild : null
    return {
      mode,
      anchor,
      anchorTop: anchor?.getBoundingClientRect().top || 0,
      scrollHeight: container.scrollHeight
    }
  }

  function removeStart(count) {
    if (!container.children[count]) return 0
    const before = container.scrollHeight
    const nodes = [...container.children].slice(0, count)
    onRemove(nodes)
    nodes.forEach(node => node.remove())
    const removedHeight = before - container.scrollHeight
    setHeight(topSpacer, height(topSpacer) + removedHeight)
    return nodes.length
  }

  function removeEnd(count) {
    const before = container.scrollHeight
    const nodes = [...container.children].slice(-count)
    onRemove(nodes)
    nodes.forEach(node => node.remove())
    setHeight(bottomSpacer, height(bottomSpacer) + before - container.scrollHeight)
    return nodes.length
  }

  function finish(state) {
    if (state.mode === 'prepend' && state.anchor?.isConnected) {
      const insertedHeight = state.anchor.getBoundingClientRect().top - state.anchorTop
      const consumed = Math.min(height(topSpacer), Math.max(0, insertedHeight))
      setHeight(topSpacer, height(topSpacer) - consumed)
      const remaining = insertedHeight - consumed
      if (remaining > 0) window.scrollBy(0, remaining)
    } else if (state.mode === 'append') {
      const insertedHeight = Math.max(0, container.scrollHeight - state.scrollHeight)
      setHeight(bottomSpacer, height(bottomSpacer) - Math.min(height(bottomSpacer), insertedHeight))
    }

    const excess = container.children.length - maximum
    if (excess <= 0) return { start: 0, end: 0 }
    const count = Math.min(container.children.length, Math.max(trim, excess))
    return state.mode === 'prepend' ? { start: 0, end: removeEnd(count) } : { start: removeStart(count), end: 0 }
  }

  return { begin, finish, reset }
}
