import { createBoundedImageLoader } from './bounded-image-loader.js?v=3'

// Covers that already loaded in this page. A list redrawn after a click puts them back at once instead of fading each one in again.
const shown = new Set()
// Covers the server said do not exist (a track with no art). A redraw does not ask for them again.
const missing = new Set()

export function createAudioArtworkLoader({ IntersectionObserverClass = globalThis.IntersectionObserver, concurrency = 4 } = {}) {
  const loader = createBoundedImageLoader({ concurrency })
  const observer = IntersectionObserverClass
    ? new IntersectionObserverClass(entries => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          observer.unobserve(entry.target)
          loader.enqueue(entry.target.querySelector('img[data-src]'))
        }
      }, { rootMargin: '240px' })
    : null

  // A tile left with nothing in it gets a placeholder (see cover-loading.css) instead of staying a blank block.
  const markFailed = element => { if (!element.children.length && !element.textContent.trim()) element.dataset.artFailed = '' }

  function load(element, url, immediate = false) {
    if (!element || !url || element.querySelector('img')) return
    if (missing.has(url)) { markFailed(element); return }
    const image = element.ownerDocument.createElement('img')
    image.alt = ''
    image.className = 'art-img'
    image.decoding = 'async'
    image.fetchPriority = immediate ? 'high' : 'low'
    image.dataset.src = url
    const again = shown.has(url)
    if (again) image.classList.add('loaded', 'again')
    image.addEventListener('load', () => { shown.add(url); image.classList.add('loaded') }, { once: true })
    image.addEventListener('error', () => {
      missing.add(url)
      image.remove()
      markFailed(element)
    }, { once: true })
    element.append(image)
    if (immediate || !observer || again) loader.enqueue(image)
    else observer.observe(element)
  }

  function release(nodes) {
    for (const node of nodes) {
      for (const element of node.querySelectorAll('[data-artwork], [data-art]'))
        observer?.unobserve(element)
      loader.release(node)
    }
  }

  return { load, release }
}
