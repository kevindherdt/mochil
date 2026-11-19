import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

// Same WCAG relative-luminance/contrast math as theme-controls.test.mjs.
function luminance(hex) {
  const [r, g, b] = [1, 3, 5].map(at => parseInt(hex.slice(at, at + 2), 16) / 255)
    .map(value => (value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}
const contrast = (a, b) => {
  const [high, low] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (high + 0.05) / (low + 0.05)
}

// Mirrors the browser's `color-mix(in srgb, A P%, B)`: linear per-channel interpolation
// of raw sRGB byte values (not gamma-corrected linear light).
function hex6(hex) {
  if (hex.length === 4) return '#' + [...hex.slice(1)].map(c => c + c).join('')
  return hex
}
function colorMix(hexA, pctA, hexB) {
  hexA = hex6(hexA); hexB = hex6(hexB)
  const a = [1, 3, 5].map(at => parseInt(hexA.slice(at, at + 2), 16))
  const b = [1, 3, 5].map(at => parseInt(hexB.slice(at, at + 2), 16))
  const mixed = a.map((v, i) => Math.round(v * pctA / 100 + b[i] * (1 - pctA / 100)))
  return '#' + mixed.map(v => v.toString(16).padStart(2, '0')).join('')
}

async function loadPaletteThemeTokens() {
  const css = await readFile(new URL('../static/theme-tokens.css', import.meta.url), 'utf8')
  const blocks = [...css.matchAll(/data-palette="(\w+)"\]\[data-theme(?:="(\w+)")?\][^{]*\{([^}]*)\}/g)]
  return blocks.map(([, palette, theme, body]) => {
    const token = name => body.match(new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{3,6})`))?.[1]
    return { palette, theme: theme || 'any', surface: hex6(token('surface')), text: hex6(token('text')) }
  })
}

test('editor badge tones (bad/warn/good) read at AA contrast against every palette and theme', async () => {
  const css = await readFile(new URL('../static/editor.css', import.meta.url), 'utf8')
  const tones = [...css.matchAll(/\.ed-badge\[data-tone=(\w+)\]\{[^}]*color:\s*([^;}]+);?[^}]*\}/g)]
    .reduce((map, [, tone, colorExpr]) => map.set(tone, colorExpr.trim()), new Map())

  for (const tone of ['bad', 'warn', 'good']) {
    const expr = tones.get(tone)
    assert.ok(expr, `.ed-badge[data-tone=${tone}] must declare a color`)
    // color-mix(in srgb, <fixedHex> <pct>%, var(--text)) or a bare fallback hex.
    const mixMatch = expr.match(/color-mix\(in srgb,\s*(?:var\([^,]+,\s*)?(#[0-9a-fA-F]{3,6})\)?\s*(\d+)%,\s*var\(--text\)\)/)
    const bareMatch = expr.match(/^var\(--\w+,\s*(#[0-9a-fA-F]{3,6})\)$/)
    assert.ok(mixMatch || bareMatch, `${tone} color must mix a fixed hue with var(--text) for theme-adaptive contrast, got: ${expr}`)

    for (const { palette, theme, surface, text } of await loadPaletteThemeTokens()) {
      const resolved = mixMatch ? colorMix(mixMatch[1], Number(mixMatch[2]), text) : bareMatch[1]
      const ratio = contrast(resolved, surface)
      assert.ok(ratio >= 4.5, `${tone} badge text on ${palette}/${theme} is only ${ratio.toFixed(2)}:1 (need >= 4.5:1), resolved ${resolved} vs surface ${surface}`)
    }
  }
})
