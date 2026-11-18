// Runs before the classic reader loads: Readium is the default reader, so hand
// the book over unless classic was chosen in Settings or asked for with ?engine=classic.
(function () {
  try {
    var asked = new URLSearchParams(location.search).get('engine')
    var engine = asked || localStorage.getItem('bobobooks-reader-engine')
    if (engine !== 'classic') {
      location.replace(location.pathname + '?engine=readium' + location.hash)
    }
  } catch (e) {}
})()
