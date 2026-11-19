// The graph view: every page as a dot, every [[link]] as a line. Drawn on a
// canvas so a few hundred pages stay smooth; drag to pan, scroll to zoom, click a page to open it.
import { selectGraph, layoutGraph } from './vault-graph-layout.js?v=3'

export async function openGraph({ pane, library, libraryName, activeId, requestJSON, escapeHTML, onOpen, onTabs = () => {} }) {
  pane.className = 'vault-pane document-reader vault-document vault-graph-pane'
  pane.innerHTML = `<header class="vault-notebar"><div class="vault-notebar-column"><nav class="vault-crumbs" aria-label="Location"><span>${escapeHTML(libraryName || 'Vault')}</span><i aria-hidden="true">/</i><span>Graph</span></nav><div class="vault-page-actions"><span class="vault-graph-note" role="status"></span><div class="vault-graph-modes" role="group" aria-label="Show"><button type="button" data-graph-mode="local">This page</button><button type="button" data-graph-mode="all">Everything</button></div></div></div></header><div class="vault-graph-host"><canvas role="img" aria-label="Graph of pages and the links between them"></canvas><p class="vault-graph-empty" hidden></p></div>`
  onTabs(pane)
  const host = pane.querySelector('.vault-graph-host'), canvas = host.querySelector('canvas')
  const note = pane.querySelector('.vault-graph-note'), empty = host.querySelector('.vault-graph-empty')
  note.textContent = 'Loading…'
  let graph
  try {
    graph = (await requestJSON(`/api/documents/${encodeURIComponent(library)}/graph`)).data
  } catch (error) {
    note.textContent = ''; empty.hidden = false; empty.textContent = error.message || 'Could not load the graph.'
    return
  }
  if (!canvas.isConnected) return

  const ctx = canvas.getContext('2d')
  const style = getComputedStyle(pane)
  const color = name => style.getPropertyValue(name).trim() || '#888'
  let mode = graph.nodes.some(node => node.id === activeId) ? 'local' : 'all'
  let shown = { nodes: [], edges: [], hidden: 0 }, spot = { x: new Float64Array(0), y: new Float64Array(0) }
  let neighbours = []
  const view = { scale: 1, x: 0, y: 0 }
  let width = 0, height = 0, hover = -1, frame = 0

  const point = index => ({ x: spot.x[index] * width * view.scale + view.x, y: spot.y[index] * height * view.scale + view.y })
  const radius = index => 3 + Math.min(9, Math.sqrt(shown.nodes[index].degree) * 1.6)

  function draw() {
    frame = 0
    const ratio = window.devicePixelRatio || 1
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0)
    ctx.clearRect(0, 0, width, height)
    const lit = hover >= 0 ? new Set([hover, ...neighbours[hover]]) : null
    ctx.lineWidth = 1
    for (const [a, b] of shown.edges) {
      const from = point(a), to = point(b)
      const on = lit && (a === hover || b === hover)
      ctx.strokeStyle = on ? color('--accent') : color('--line')
      ctx.globalAlpha = lit && !on ? 0.35 : 1
      ctx.beginPath(); ctx.moveTo(from.x, from.y); ctx.lineTo(to.x, to.y); ctx.stroke()
    }
    ctx.font = '12px "Source Sans","Segoe UI",sans-serif'
    ctx.textAlign = 'center'
    shown.nodes.forEach((node, index) => {
      const at = point(index)
      if (at.x < -20 || at.y < -20 || at.x > width + 20 || at.y > height + 20) return
      const current = node.id === activeId
      ctx.globalAlpha = lit && !lit.has(index) ? 0.3 : 1
      ctx.fillStyle = current || index === hover ? color('--accent') : color('--muted')
      ctx.beginPath(); ctx.arc(at.x, at.y, radius(index), 0, Math.PI * 2); ctx.fill()
      const label = current || index === hover || lit?.has(index) || view.scale > 1.6 || node.degree >= 4 || shown.nodes.length <= 40
      if (label) {
        ctx.fillStyle = color('--text')
        ctx.fillText(node.title.length > 28 ? `${node.title.slice(0, 27)}…` : node.title, at.x, at.y + radius(index) + 13)
      }
    })
    ctx.globalAlpha = 1
  }
  const redraw = () => { if (!frame) frame = requestAnimationFrame(draw) }

  function size() {
    const box = host.getBoundingClientRect()
    width = Math.max(200, Math.floor(box.width)); height = Math.max(200, Math.floor(box.height))
    const ratio = window.devicePixelRatio || 1
    canvas.width = width * ratio; canvas.height = height * ratio
    canvas.style.width = `${width}px`; canvas.style.height = `${height}px`
    redraw()
  }

  async function show(next) {
    mode = next
    pane.querySelectorAll('[data-graph-mode]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.graphMode === mode)))
    note.textContent = 'Arranging…'
    await new Promise(resolve => requestAnimationFrame(resolve))
    if (!canvas.isConnected) return
    shown = selectGraph(graph, { activeId, mode })
    spot = layoutGraph(shown.nodes, shown.edges)
    neighbours = shown.nodes.map(() => [])
    for (const [a, b] of shown.edges) { neighbours[a].push(b); neighbours[b].push(a) }
    Object.assign(view, { scale: 1, x: 0, y: 0 })
    // The hovered index belonged to the previous graph; keeping it would spread an
    // undefined neighbour list and paint a node that is not there.
    hover = -1
    drag = null
    empty.hidden = shown.nodes.length > 0
    empty.textContent = 'No pages link to each other yet. Type [[ in a page to link it to another.'
    note.textContent = shown.nodes.length ? `${shown.nodes.length.toLocaleString()} pages${shown.hidden ? ` · ${shown.hidden.toLocaleString()} not shown` : ''}` : ''
    size()
  }

  const nearest = (x, y) => {
    let best = -1, gap = Infinity
    shown.nodes.forEach((_node, index) => {
      const at = point(index), distance = Math.hypot(at.x - x, at.y - y)
      if (distance <= radius(index) + 5 && distance < gap) { best = index; gap = distance }
    })
    return best
  }
  const local = event => { const box = canvas.getBoundingClientRect(); return { x: event.clientX - box.left, y: event.clientY - box.top } }

  let drag = null
  canvas.addEventListener('pointerdown', event => { drag = { ...local(event), moved: false, vx: view.x, vy: view.y }; canvas.setPointerCapture(event.pointerId) })
  canvas.addEventListener('pointermove', event => {
    const at = local(event)
    if (drag) {
      if (Math.hypot(at.x - drag.x, at.y - drag.y) > 3) drag.moved = true
      if (drag.moved) { view.x = drag.vx + at.x - drag.x; view.y = drag.vy + at.y - drag.y; redraw() }
      return
    }
    const over = nearest(at.x, at.y)
    if (over !== hover) { hover = over; canvas.style.cursor = over >= 0 ? 'pointer' : 'grab'; redraw() }
  })
  canvas.addEventListener('pointerup', event => {
    const click = drag && !drag.moved
    drag = null
    if (!click) return
    const at = local(event), index = nearest(at.x, at.y)
    if (index >= 0) onOpen(shown.nodes[index])
  })
  canvas.addEventListener('wheel', event => {
    event.preventDefault()
    const at = local(event), factor = event.deltaY < 0 ? 1.15 : 1 / 1.15
    const next = Math.min(8, Math.max(0.4, view.scale * factor))
    const k = next / view.scale
    view.x = at.x - (at.x - view.x) * k; view.y = at.y - (at.y - view.y) * k; view.scale = next
    redraw()
  }, { passive: false })
  canvas.addEventListener('dblclick', () => { Object.assign(view, { scale: 1, x: 0, y: 0 }); redraw() })
  pane.querySelector('.vault-graph-modes').addEventListener('click', event => {
    const next = event.target.closest('[data-graph-mode]')?.dataset.graphMode
    if (next && next !== mode) show(next)
  })
  new ResizeObserver(() => { if (canvas.isConnected) size() }).observe(host)
  await show(mode)
}
