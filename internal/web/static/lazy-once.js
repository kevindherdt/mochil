// Loads a feature module once, but never remembers a failure: a dropped
// connection while fetching one script must not break that feature until the
// next full reload. The next request simply tries again.
export function lazyOnce(load) {
  let value;
  let pending;
  return () => {
    if (value !== undefined) return Promise.resolve(value);
    pending ||= load().then(
      (result) => (value = result),
      (error) => {
        pending = null;
        throw error;
      },
    );
    return pending;
  };
}

// Tells the person which feature could not load, in the shared status toast.
export function reportLoadFailure(document, name, error) {
  if (error) console.error(`${name} failed`, error);
  const status = document.querySelector("#status");
  if (status) status.textContent = `${name} could not load. Check your connection and try again.`;
}
