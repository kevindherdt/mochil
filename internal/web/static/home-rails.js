// Home's shelves ask the server for overlapping lists -- the held shelf and Jump back in
// both want reading progress -- so one in-flight request per URL turns that into a single
// call each, and one object owns them so the wiring does not grow the app shell.

export function createHomeRails({
  document,
  $,
  $$,
  escapeHTML,
  releaseCovers,
  installCoverFallbacks,
  isLibraryVisible,
  createJumpBackIn,
  createHeldRail,
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
  const jumpBackIn = createJumpBackIn(shared);
  const held = createHeldRail(shared);

  // Opening Home refreshes them together, as one action.
  return { jumpBackIn, held, load: () => { jumpBackIn.load(); held.load(); } };
}
