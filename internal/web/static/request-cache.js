// Home asks for the same lists from more than one rail — Continue reading and
// Finished both want /api/reading?finished=…, and Jump back in wants the video
// and audio progress the same screen already fetched. Sharing one in-flight
// GET per URL turns that into a single request; the entry is dropped the moment
// it settles, so every later load still reads fresh state.
export function createRequestCache(requestJSON) {
  const inflight = new Map();

  function json(url, options) {
    // Only a read can be shared: two writes to the same address are two
    // different operations even though the URL matches.
    if (String(options?.method || "GET").toUpperCase() !== "GET")
      return Promise.resolve(requestJSON(url, options));
    // A signal belongs to the caller that passed it. Sharing such a request let
    // one rail's abort cancel every other consumer of the same URL, and made the
    // second consumer's own abort do nothing.
    if (options?.signal) return Promise.resolve(requestJSON(url, options))
    const pending = inflight.get(url)
    if (pending) return pending
    let request;
    try {
      request = Promise.resolve(requestJSON(url, options));
    } catch (error) {
      return Promise.reject(error);
    }
    inflight.set(url, request);
    const settle = () => {
      if (inflight.get(url) === request) inflight.delete(url);
    };
    request.then(settle, settle);
    return request;
  }

  return { json, clear: () => inflight.clear() };
}
