// Readium is the default reader. A browser that chose the classic reader in
// Settings is sent there before anything loads.
(function () {
  try {
    if (new URLSearchParams(location.search).get('engine')) return
    if (localStorage.getItem('bobobooks-reader-engine') === 'classic') {
      location.replace(location.pathname + '?engine=classic' + location.hash)
    }
  } catch (e) {}
})()
