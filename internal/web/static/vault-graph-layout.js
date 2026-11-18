// The shape of the link graph: which pages to draw and where to put them.
// Pure functions, so they can be tested without a canvas.

/**
 * Picks the part of the graph worth drawing. "local" is the open page and the
 * pages within two links of it; "all" is the best-connected pages, up to `cap` (unlinked pages fill what is left).
 * Returns nodes (with a link count), edges as index pairs, and how many pages were left out.
 */
export function selectGraph(graph, { activeId = '', mode = 'all', cap = 400 } = {}) {
  const degree = new Array(graph.nodes.length).fill(0)
  const neighbours = graph.nodes.map(() => [])
  for (const [from, to] of graph.edges) {
    degree[from]++; degree[to]++
    neighbours[from].push(to); neighbours[to].push(from)
  }
  let keep
  const start = graph.nodes.findIndex(node => node.id === activeId)
  if (mode === 'local' && start >= 0) {
    keep = new Set([start])
    let ring = [start]
    for (let depth = 0; depth < 2; depth++) {
      const next = []
      for (const index of ring) for (const other of neighbours[index]) if (!keep.has(other)) { keep.add(other); next.push(other) }
      ring = next
    }
  } else {
    const ranked = graph.nodes.map((_, index) => index)
    ranked.sort((a, b) => degree[b] - degree[a] || a - b)
    keep = new Set(ranked.slice(0, cap))
    if (start >= 0) keep.add(start)
  }
  const order = [...keep].sort((a, b) => a - b)
  const remap = new Map(order.map((index, position) => [index, position]))
  return {
    nodes: order.map(index => ({ ...graph.nodes[index], degree: degree[index] })),
    edges: graph.edges.filter(([a, b]) => remap.has(a) && remap.has(b)).map(([a, b]) => [remap.get(a), remap.get(b)]),
    hidden: graph.nodes.length - order.length,
  }
}

/** Force-directed positions in the unit square. Deterministic: the same graph always lands the same way. */
export function layoutGraph(nodes, edges, options = {}) {
  const count = nodes.length
  // Every pair of pages pushes on each other, so big graphs get fewer, still-settling rounds.
  const iterations = options.iterations ?? Math.max(70, Math.min(220, Math.round(220 * 150 / Math.max(count, 150))))
  const x = new Float64Array(count), y = new Float64Array(count)
  for (let index = 0; index < count; index++) { // a spiral, so nothing starts on top of anything else
    const angle = index * 2.399963, radius = 0.45 * Math.sqrt((index + 0.5) / Math.max(count, 1))
    x[index] = 0.5 + radius * Math.cos(angle)
    y[index] = 0.5 + radius * Math.sin(angle)
  }
  if (count < 2) return { x, y }
  const ideal = 0.9 / Math.sqrt(count)
  const dx = new Float64Array(count), dy = new Float64Array(count)
  for (let step = 0; step < iterations; step++) {
    dx.fill(0); dy.fill(0)
    for (let a = 0; a < count; a++) {
      for (let b = a + 1; b < count; b++) {
        let ex = x[a] - x[b], ey = y[a] - y[b]
        let distance = Math.hypot(ex, ey)
        if (distance < 1e-4) { ex = (a - b) * 1e-3; ey = 1e-3; distance = Math.hypot(ex, ey) }
        const push = (ideal * ideal) / distance
        dx[a] += (ex / distance) * push; dy[a] += (ey / distance) * push
        dx[b] -= (ex / distance) * push; dy[b] -= (ey / distance) * push
      }
    }
    for (const [a, b] of edges) {
      const ex = x[a] - x[b], ey = y[a] - y[b]
      const distance = Math.hypot(ex, ey) || 1e-4
      const pull = (distance * distance) / ideal
      dx[a] -= (ex / distance) * pull; dy[a] -= (ey / distance) * pull
      dx[b] += (ex / distance) * pull; dy[b] += (ey / distance) * pull
    }
    const cool = 0.1 * (1 - step / iterations) + 0.002
    for (let index = 0; index < count; index++) {
      dx[index] -= (x[index] - 0.5) * 0.6 * ideal; dy[index] -= (y[index] - 0.5) * 0.6 * ideal // gravity
      const length = Math.hypot(dx[index], dy[index]) || 1
      const move = Math.min(length, cool)
      x[index] += (dx[index] / length) * move
      y[index] += (dy[index] / length) * move
    }
  }
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity
  for (let index = 0; index < count; index++) {
    minX = Math.min(minX, x[index]); maxX = Math.max(maxX, x[index]); minY = Math.min(minY, y[index]); maxY = Math.max(maxY, y[index])
  }
  const span = Math.max(maxX - minX, maxY - minY) || 1
  for (let index = 0; index < count; index++) {
    x[index] = 0.5 + (x[index] - (minX + maxX) / 2) / span * 0.92
    y[index] = 0.5 + (y[index] - (minY + maxY) / 2) / span * 0.92
  }
  return { x, y }
}
