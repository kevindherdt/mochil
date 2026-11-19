let loader

export function loadHLSClient() {
  if (window.Hls) return Promise.resolve(window.Hls)
  if (loader) return loader
  loader = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = '/assets/vendor/hls.js/hls.light.min.js?v=1'
    script.onload = () => window.Hls ? resolve(window.Hls) : reject(new Error('HLS client did not initialize'))
    script.onerror = () => reject(new Error('HLS client could not be loaded'))
    document.head.append(script)
  })
  return loader
}
