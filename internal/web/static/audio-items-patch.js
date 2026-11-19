// Puts new markup on the shelf. Music Discover is rebuilt on every state change (playing a song moves it into
// "Recently played"), and replacing all of it reloads every cover on the page. Here only the sections whose markup
// changed are swapped, so the rest keep their covers, their sideways scroll position and their open menus.
const signatures = new WeakMap() // the markup a section was built from; the page itself changes it (covers load, fade in)

// changedSections says which sections differ, or null when the shape changed and everything must be replaced.
export function changedSections(before, after) {
  if (before.length !== after.length) return null
  return after.flatMap((signature, index) => (signature === before[index] ? [] : [index]))
}

const discovery = element => !!element?.classList?.contains('music-discovery')

// Any other list is compared the same way, row by row (and one level inside a wrapper such as a cover grid): a row whose
// markup did not change is left alone, so playing a song does not reload every cover in Favorites or a playlist.
const DEPTH = 2
function remember(element, depth = 0) {
  signatures.set(element, element.outerHTML)
  if (depth < DEPTH) for (const child of element.children) remember(child, depth + 1)
}
function syncAttributes(old, fresh) {
  for (const { name } of [...old.attributes]) if (!fresh.hasAttribute(name)) old.removeAttribute(name)
  for (const { name, value } of [...fresh.attributes]) if (old.getAttribute(name) !== value) old.setAttribute(name, value)
}
export function morphChildren(parent, fresh, depth = 0) {
  const older = [...parent.children], newer = [...fresh.children]
  if (older.length !== newer.length || parent.childNodes.length !== fresh.childNodes.length) {
    parent.replaceChildren(...fresh.childNodes)
    for (const child of parent.children) remember(child, depth + 1)
    return
  }
  newer.forEach((next, index) => {
    const old = older[index], signature = next.outerHTML
    if (signatures.get(old) === signature) return
    if (depth + 1 < DEPTH && old.tagName === next.tagName && old.children.length && next.children.length) {
      syncAttributes(old, next)
      morphChildren(old, next, depth + 1)
      signatures.set(old, signature)
    } else {
      old.replaceWith(next)
      remember(next, depth + 1)
    }
  })
}

export function patchItems(container, html) {
  const template = container.ownerDocument.createElement('template')
  template.innerHTML = html
  const old = container.firstElementChild, fresh = template.content.firstElementChild
  if (container.childElementCount === 1 && template.content.childElementCount === 1 && discovery(old) && discovery(fresh)) {
    const fresher = [...fresh.children], older = [...old.children]
    const changed = changedSections(older.map(section => signatures.get(section)), fresher.map(section => section.outerHTML))
    if (changed) {
      for (const index of changed) { signatures.set(fresher[index], fresher[index].outerHTML); older[index].replaceWith(fresher[index]) }
      return
    }
  }
  if (discovery(container.firstElementChild) || discovery(fresh)) {
    container.innerHTML = html
    if (discovery(container.firstElementChild)) for (const section of container.firstElementChild.children) signatures.set(section, section.outerHTML)
    return
  }
  if (container.children.length && fresh) morphChildren(container, template.content)
  else { container.innerHTML = html; for (const child of container.children) remember(child, 1) }
}
