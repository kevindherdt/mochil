import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('native select menus inherit explicit theme colors', async () => {
  const css = await readFile(new URL('../static/app.css', import.meta.url), 'utf8')
  assert.match(css, /select (?:option\s*,\s*select optgroup|optgroup\s*,\s*select option)\s*\{\s*background:\s*var\(--surface\);\s*color:\s*var\(--text\)/)
})

test('the default palette sits in the middle of the range, with its own light twin', async () => {
  const css = await readFile(new URL('../static/theme-tokens.css', import.meta.url), 'utf8')
  assert.match(css, /data-palette="mochil".*data-theme="dark"/)
  // A neutral mid rather than a near-black: the page must keep working when the device asks for either mode.
  assert.match(css, /--bg:\s*#443c34/)
  assert.match(css, /--surface:\s*#504740/)
  assert.match(css, /data-palette="mochil".*data-theme="light"/)
  assert.match(css, /--bg:\s*#fdf1e3/)
  assert.match(css, /--surface:\s*#fffaf3/)
  assert.match(css, /--accent:\s*#b8420a/)
})

// One palette carries both a dark and a light block, so the device's switch still means something; Paper keeps
// one set of values, and so does e-ink.
const palettes = [
  { id: 'mochil', modes: ['dark', 'light'] },
  { id: 'kiln', modes: ['dark', 'light'] },
  { id: 'magenta-cream', modes: ['dark', 'light'] },
  // e-ink is a screen, not a mood: pure black on white is the point of it.
  { id: 'eink', modes: ['dark', 'light'], pure: true },
]

function luminance(hex) {
  const [r, g, b] = [1, 3, 5].map(at => parseInt(hex.slice(at, at + 2), 16) / 255)
    .map(value => (value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}
const contrast = (a, b) => {
  const [high, low] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (high + 0.05) / (low + 0.05)
}
function hue(hex) {
  const [r, g, b] = [1, 3, 5].map(at => parseInt(hex.slice(at, at + 2), 16) / 255)
  const max = Math.max(r, g, b), min = Math.min(r, g, b)
  if (max === min) return null
  const d = max - min
  const value = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4
  return (value * 60 + 360) % 360
}

test('every theme is registered, readable, and never green', async () => {
  const css = await readFile(new URL('../static/theme-tokens.css', import.meta.url), 'utf8')
  const controller = await readFile(new URL('../static/theme-controller.js', import.meta.url), 'utf8')
  const settings = await readFile(new URL('../static/theme-settings-template.js', import.meta.url), 'utf8')
  for (const palette of palettes) {
    assert.match(controller, new RegExp(`"${palette.id}"`), `${palette.id} must be an accepted palette`)
    assert.match(settings, new RegExp(`value="${palette.id}"`), `${palette.id} must be offered in Appearance`)
    for (const mode of palette.modes) {
      const selector = mode === 'any'
        ? `data-palette="${palette.id}"\\]\\[data-theme\\]`
        : `data-palette="${palette.id}"\\]\\[data-theme="${mode}"\\]`
      const block = css.match(new RegExp(`${selector}[^}]*\\{([^}]*)\\}`))?.[1]
      assert.ok(block, `${palette.id} ${mode} is missing`)
      // #000 is how the e-ink block writes pure black; expand short hex rather than skipping the palette.
      const token = name => {
        const raw = block.match(new RegExp(`--${name}:\\s*#([0-9a-fA-F]{6}|[0-9a-fA-F]{3})`))?.[1]
        return raw ? `#${raw.length === 3 ? raw.split('').map((c) => c + c).join('') : raw}` : undefined
      }
      const [bg, surface, text, muted, accent, ink] = ['bg', 'surface', 'text', 'muted', 'accent', 'accent-ink'].map(token)
      for (const [label, value] of Object.entries({ bg, surface, text, muted, accent, ink })) assert.ok(value, `${palette.id} ${mode} lacks ${label}`)
      assert.ok(contrast(text, bg) >= 7, `${palette.id} ${mode} text contrast`)
      assert.ok(contrast(muted, bg) >= 4.5 && contrast(muted, surface) >= 4.5, `${palette.id} ${mode} secondary text contrast`)
      assert.ok(contrast(accent, bg) >= 4.5, `${palette.id} ${mode} accent contrast`)
      assert.ok(contrast(ink, accent) >= 4.5, `${palette.id} ${mode} text on accent contrast`)
      if (!palette.pure) assert.doesNotMatch(bg.toLowerCase(), /^#(000000|ffffff)$/, `${palette.id} ${mode} must not be pure black or white`)
      for (const color of [bg, surface, accent]) {
        const angle = hue(color)
        assert.ok(angle === null || angle < 70 || angle > 170, `${palette.id} ${mode} uses a green (${color})`)
      }
    }
  }
})

test('the Appearance picker offers four themes, and every one is defined in the shared token file', async () => {
  const css = await readFile(new URL('../static/theme-tokens.css', import.meta.url), 'utf8')
  const controller = await readFile(new URL('../static/theme-controller.js', import.meta.url), 'utf8')
  const settings = await readFile(new URL('../static/theme-settings-template.js', import.meta.url), 'utf8')
  const offered = [...controller.match(/const palettes = new Set\(\[([^\]]*)\]/)[1].matchAll(/"([a-z-]+)"/g)].map(match => match[1])
  // Offered order is the picker's own order, so compare the sorted set: "mochil" now sorts after
  // "kiln", where the old product name sorted first.
  assert.deepEqual(offered.sort(), ['eink', 'kiln', 'magenta-cream', 'mochil'])
  for (const palette of offered) {
    assert.match(settings, new RegExp(`value="${palette}"`), `${palette} is missing from the picker`)
    assert.match(css, new RegExp(`data-palette="${palette}"\\]\\[data-theme`), `${palette} is not in theme-tokens.css`)
  }
  assert.doesNotMatch(css, /nocturne/i, 'the dead Nocturne token block must be gone')
  assert.doesNotMatch(settings, /nocturne/i)
  for (const removed of ['nocturne', 'dusk', 'twilight', 'tide', 'rosewood', 'ember', 'manjaro', 'ubuntu', 'material']) {
    assert.doesNotMatch(settings, new RegExp(`value="${removed}"`), `${removed} must not be offered`)
  }
})
