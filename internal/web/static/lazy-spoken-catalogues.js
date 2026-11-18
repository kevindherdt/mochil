function createLazyCatalogue({ load, empty, methods, resetMethods }) {
  let catalogue;
  let loading;
  let generation = 0;

  async function ensure() {
    if (catalogue) return catalogue;
    loading ||= load().then(create => (catalogue = create()));
    return loading;
  }

  const facade = {};
  for (const [name, fallback] of Object.entries(empty)) {
    Object.defineProperty(facade, name, { enumerable: true, get: () => catalogue?.[name] ?? fallback });
  }
  for (const name of methods) facade[name] = async (...args) => {
    const requestedGeneration = generation;
    const target = await ensure();
    if (requestedGeneration !== generation) return false;
    return target[name](...args);
  };
  for (const name of resetMethods) facade[name] = () => {
    generation += 1;
    catalogue?.[name]();
  };
  return facade;
}

export function createLazyPodcastCatalogue(requestJSON) {
  return createLazyCatalogue({
    load: async () => {
      const { createPodcastCatalogue } = await import('./podcast-catalogue.js?v=2');
      return () => createPodcastCatalogue({ requestJSON });
    },
    empty: { show: '', episodes: [], total: 0, loading: false },
    methods: ['loadShow'],
    resetMethods: ['abort', 'reset', 'closeShow'],
  });
}

export function createLazyAudiobookCatalogue(requestJSON, pageSize) {
  return createLazyCatalogue({
    load: async () => {
      const { createAudiobookCatalogue } = await import('./audiobook-catalogue.js?v=3');
      return () => createAudiobookCatalogue({ requestJSON, pageSize });
    },
    empty: { group: '', series: [], books: [], total: 0, loading: false, groupType: 'series' },
    methods: ['loadSeries', 'loadAuthors', 'loadGroup'],
    resetMethods: ['abort', 'reset'],
  });
}
