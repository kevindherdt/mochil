// Home's three shelves ask the server for overlapping lists: Continue reading and
// Finished both read /api/reading, and Jump back in wants the video and audio
// progress the same screen already fetched. One in-flight request per URL turns
// that into a single call each, and one object owns the three so the wiring does
// not grow the app shell.

export function createHomeRails({
  document,
  $,
  $$,
  escapeHTML,
  releaseCovers,
  installCoverFallbacks,
  isLibraryVisible,
  createReadingDashboard,
  createJumpBackIn,
  createFinishedRail,
  requests,
}) {
  const shared = {
    document,
    $,
    escapeHTML,
    requestJSON: requests.json,
    releaseCovers,
    installCoverFallbacks,
    isLibraryVisible,
  };
  const reading = createReadingDashboard({ ...shared, $$ });
  const jumpBackIn = createJumpBackIn(shared);
  const finished = createFinishedRail(shared);

  // Opening Home refreshes all three together, as one action.
  const loadReading = reading.load;
  reading.load = (...args) => {
    jumpBackIn.load();
    finished.load();
    return loadReading(...args);
  };
  return { reading, jumpBackIn, finished };
}
