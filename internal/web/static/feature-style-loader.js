const pending = new Map();

/** Loads route-owned styles once and resolves only after they are applied. */
export function loadFeatureStyles(document, hrefs) {
  return Promise.all(hrefs.map(href => loadStyle(document, href)));
}

function loadStyle(document, href) {
  const existing = [...document.styleSheets].find(sheet => sheet.href && new URL(sheet.href).pathname === new URL(href, document.baseURI).pathname);
  if (existing) return Promise.resolve();
  if (pending.has(href)) return pending.get(href);
  const promise = new Promise((resolve, reject) => {
    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = href;
    link.dataset.featureStyle = href;
    link.onload = resolve;
    link.onerror = () => reject(new Error(`Could not load interface styles: ${href}`));
    document.head.append(link);
  });
  pending.set(href, promise);
  return promise;
}
