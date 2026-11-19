export function createLibraryLoader({
  requestJSON,
  libraryVisibility,
  readingDashboard,
  videoLibrary,
  audioLibrary,
  libraryState,
  renderToolbar,
  renderAlphabet,
  isLibraryView,
  loadCurrentView,
  setStatus,
  onFirstRun = () => {},
  showProblem = () => {},
  clearProblem = () => {},
  onReady = () => {},
  retryDelays = [2000, 5000, 15000, 30000, 60000],
  requestTimeout = 10000,
  window: win = globalThis.window,
  document: doc = globalThis.document,
}) {
  let retryTimer, inFlight, pendingRefresh = false, disposed = false, failures = 0, ready = false, failed = false, authFailure = false;
  function scheduleRetry() {
    if (disposed || retryTimer || authFailure || !retryDelays.length || doc?.hidden) return;
    retryTimer = setTimeout(() => { retryTimer = null; load(); }, retryDelays[Math.min(failures - 1, retryDelays.length - 1)]);
  }
  function wake() {
    if (doc?.hidden) { clearTimeout(retryTimer); retryTimer = null; return; }
    if (!failed || disposed || authFailure) return;
    clearTimeout(retryTimer);
    retryTimer = null;
    failures = 0;
    load();
  }
  function dispose() {
    disposed = true;
    clearTimeout(retryTimer);
    win?.removeEventListener("online", wake);
    doc?.removeEventListener("visibilitychange", wake);
    win?.removeEventListener("pagehide", pagehide);
  }
  function pagehide(event) { if (!event.persisted) dispose(); }
  win?.addEventListener("online", wake);
  win?.addEventListener("pagehide", pagehide);
  doc?.addEventListener("visibilitychange", wake);
  const auxiliary = [
    { run: () => readingDashboard.load() },
    { run: () => videoLibrary.loadContinue?.() },
    { run: () => audioLibrary.loadContinue?.() },
  ];
  function refreshAuxiliary(task) {
    if (disposed) return;
    if (task.running) { task.pending = true; return; }
    task.running = true;
    let result;
    try { result = task.run(); } catch { task.running = false; return; }
    if (!result || typeof result.then !== "function") { task.running = false; return; }
    Promise.resolve(result).catch(() => {}).finally(() => {
      task.running = false;
      if (task.pending && !disposed) { task.pending = false; refreshAuxiliary(task); }
    });
  }
  function refreshDashboard() {
    auxiliary.forEach(refreshAuxiliary);
  }

  function applyVisible(visibleLibraries) {
    libraryState.setVisible(visibleLibraries);
    libraryState.ensureVisible();
    renderToolbar();
    renderAlphabet();
    refreshDashboard();
    if (isLibraryView()) loadCurrentView();
  }

  function load() {
    if (disposed) return Promise.resolve();
    if (inFlight) { pendingRefresh = true; return inFlight; }
    clearTimeout(retryTimer);
    retryTimer = null;
    inFlight = (async () => {
      do {
        clearTimeout(retryTimer);
        retryTimer = null;
        pendingRefresh = false;
        await performLoad();
      } while (pendingRefresh && !disposed);
    })().finally(() => { inFlight = null; });
    return inFlight;
  }

  async function performLoad() {
    clearProblem();
    let timeout;
    const controller = new AbortController();
    try {
      const result = await Promise.race([
        requestJSON("/api/libraries", { signal: controller.signal }),
        new Promise((_, reject) => { timeout = setTimeout(() => { controller.abort(); reject(new Error("Library request timed out")); }, requestTimeout); }),
      ]);
      clearTimeout(timeout);
      if (disposed) return;
      libraryState.setAll(result.data);
      libraryState.setVisible(libraryVisibility.setLibraries(result.data));
      libraryState.initializeSelection();
      libraryState.ensureVisible();
      refreshDashboard();
      renderToolbar();
      renderAlphabet();
      failed = false;
      authFailure = false;
      failures = 0;
      let viewLoaded = false;
      if (!ready) { ready = true; viewLoaded = onReady() === true; }
      if (result.data.length === 0) {
        onFirstRun();
        return;
      }
      if (!viewLoaded && isLibraryView()) await loadCurrentView();
    } catch (error) {
      if (disposed) return;
      failed = true;
      authFailure = error?.status === 401 || error?.status === 403;
      failures++;
      scheduleRetry();
      if (libraryState.all().length === 0) {
        setStatus("");
        showProblem(error, load);
      } else setStatus(`Could not open the library: ${error.message}`);
    } finally {
      clearTimeout(timeout);
    }
  }

  return { applyVisible, load, dispose };
}
