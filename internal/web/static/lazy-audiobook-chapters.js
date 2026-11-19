/** Keeps audiobook-only chapter parsing and DOM bindings out of cold Music. */
export function createLazyAudiobookChapters(options, loadModule = () => import('./audiobook-chapters.js?v=2')) {
  let chapters;
  let loading;
  let generation = 0;

  async function ensure() {
    if (chapters) return chapters;
    loading ||= Promise.resolve().then(loadModule)
      .then(({ createAudiobookChapters }) => (chapters = createAudiobookChapters(options)))
      .catch(error => { loading = undefined; throw error; });
    return loading;
  }

  return {
    current: () => chapters?.current(),
    update: () => chapters?.update(),
    openItem(item) {
      generation += 1;
      if (chapters) return chapters.openItem(item);
      if (item?.library_kind !== 'audiobooks' || !item?.chapters?.length) return;
      const requestedGeneration = generation;
      void ensure().then(value => {
        if (requestedGeneration === generation) value.openItem(item);
      }).catch(() => {
        if (requestedGeneration === generation) options.onError?.('Chapters could not load. Reopen this audiobook to retry.');
      });
    },
    close() {
      generation += 1;
      chapters?.close();
    },
  };
}
