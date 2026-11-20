// Type-ahead for a text field: as letters are typed, names that contain them are listed with how many books use
// each, so a near-duplicate spelling shows up next to the right one. Arrow keys and Enter pick, Escape closes.

// Names that begin with what was typed come before names that only contain it; each group keeps its order.
export function startsWithFirst(items, text) {
  const start = text.toLowerCase()
  return [...items.filter(item => item.name.toLowerCase().startsWith(start)), ...items.filter(item => !item.name.toLowerCase().startsWith(start))]
}

// A field that holds several values ("A & B", "tag one, tag two") suggests for the one being typed: the last.
export function lastSegment(value, separator) {
  if (!separator) return value.trim()
  return value.slice(value.lastIndexOf(separator) + 1).trim()
}

export function withLastSegment(value, separator, name) {
  if (!separator) return name
  const at = value.lastIndexOf(separator)
  return at < 0 ? name : `${value.slice(0, at + 1)} ${name}`
}

// floating puts the list on the page (not inside the field's container), for fields in a table that scrolls.
export function attachSuggestions({ input, fetchItems, separator = '', floating = false, delay = 150, document = input.ownerDocument }) {
  const list = document.createElement('ul')
  list.className = floating ? 'field-suggestions floating' : 'field-suggestions'
  list.setAttribute('role', 'listbox')
  list.hidden = true
  input.setAttribute('autocomplete', 'off')
  input.setAttribute('role', 'combobox')
  input.setAttribute('aria-expanded', 'false')
  if (!floating) input.parentElement.append(list)
  let timer = 0, serial = 0, active = -1

  const close = () => {
    list.hidden = true; list.replaceChildren(); active = -1
    input.setAttribute('aria-expanded', 'false')
    if (floating) list.remove()
  }
  const mark = index => {
    active = index
    ;[...list.children].forEach((item, at) => item.setAttribute('aria-selected', String(at === index)))
  }
  const pick = name => {
    input.value = withLastSegment(input.value, separator, name)
    close()
    input.dispatchEvent(new Event('input', { bubbles: true }))
  }
  const place = () => {
    const box = input.getBoundingClientRect()
    Object.assign(list.style, { left: `${box.left}px`, top: `${box.bottom + 2}px`, width: `${Math.max(box.width, 220)}px` })
  }
  const show = items => {
    list.replaceChildren(...items.map(({ name, count }) => {
      const item = document.createElement('li')
      item.setAttribute('role', 'option')
      const label = document.createElement('span'); label.textContent = name
      const total = document.createElement('small'); total.textContent = String(count)
      item.append(label, total)
      // pointerdown, not click: the field must not lose focus (and close the list) before the choice lands
      item.addEventListener('pointerdown', event => { event.preventDefault(); pick(name) })
      return item
    }))
    list.hidden = items.length === 0
    input.setAttribute('aria-expanded', String(items.length > 0))
    active = -1
    if (floating && items.length) { document.body.append(list); place() }
  }
  const refresh = () => {
    clearTimeout(timer)
    const text = lastSegment(input.value, separator)
    if (!text) return close()
    timer = setTimeout(async () => {
      const mine = ++serial
      let items = []
      try { items = await fetchItems(text) } catch { /* no suggestions is fine */ }
      if (mine === serial && document.activeElement === input) show(startsWithFirst(items.filter(item => item.name.toLowerCase() !== text.toLowerCase() || items.length > 1), text))
    }, delay)
  }
  input.addEventListener('input', refresh)
  input.addEventListener('blur', () => { serial++; clearTimeout(timer); setTimeout(close, 0) })
  // These run before the field's own handlers: Enter picks a suggestion instead of saving the edit, and the first
  // Escape closes the list instead of cancelling the edit or closing the dialog around it.
  input.addEventListener('keydown', event => {
    if (list.hidden) return
    const count = list.children.length
    if (event.key === 'ArrowDown') { event.preventDefault(); mark((active + 1) % count) }
    else if (event.key === 'ArrowUp') { event.preventDefault(); mark((active - 1 + count) % count) }
    else if (event.key === 'Enter' && active >= 0) { event.preventDefault(); event.stopImmediatePropagation(); pick(list.children[active].firstChild.textContent) }
    else if (event.key === 'Escape') { event.preventDefault(); event.stopImmediatePropagation(); close() }
  })
  return { close }
}
