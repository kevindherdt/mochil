function appendSeasonItems(items, knownIDs, incoming) {
  let appended = 0;
  for (const item of incoming) {
    if (knownIDs.has(item.id)) continue;
    knownIDs.add(item.id);
    items.push(item);
    appended++;
  }
  return appended;
}

export function isCurrentVideoSeasonView(view, request, items, knownIDs) {
  return (
    request === view.request &&
    items === view.items &&
    knownIDs === view.knownIDs &&
    view.target.isConnected
  );
}

export function applySeasonItems(view, request, items, knownIDs, incoming) {
  if (!isCurrentVideoSeasonView(view, request, items, knownIDs)) return false;
  appendSeasonItems(items, knownIDs, incoming);
  return true;
}
