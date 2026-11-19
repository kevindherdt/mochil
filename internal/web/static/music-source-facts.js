export function musicSourceFacts(item) {
  const facts = []
  const format = String(item.container || item.codec || '').trim().toUpperCase()
  if (format) facts.push(format)
  const bitDepth = Number(item.bit_depth)
  if (bitDepth > 0) facts.push(`${bitDepth}-bit`)
  const sampleRate = Number(item.sample_rate)
  if (sampleRate >= 1000) facts.push(`${sampleRate % 1000 ? (sampleRate / 1000).toFixed(1) : sampleRate / 1000} kHz`)
  return facts.join(' · ')
}
