// The shape of a library card and the small helpers that keep its fields
// consistent. Kept apart from library-settings.js so both it and the folder
// detection module can use them without importing the whole settings page.

export function defaultPreferredFormats(kind) {
  return ["comics", "manga"].includes(kind)
    ? ["cbz", "pdf", "epub"]
    : ["epub", "pdf", "mobi", "azw3"];
}

export function pathGet(object, path) {
  return path.split(".").reduce((value, key) => value?.[key], object);
}

export function pathSet(object, path, value) {
  const keys = path.split(".");
  let target = object;
  keys.slice(0, -1).forEach((key) => (target = target[key] ??= {}));
  target[keys.at(-1)] = value;
}

export function sameFormats(left, right) {
  return (
    Array.isArray(left) &&
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}

export function normalizePreferredFormats(value) {
  return String(value || "")
    .split(",")
    .map((format) => format.trim().replace(/^\./, "").toLowerCase())
    .filter(Boolean);
}

// One card starts from these values; the folder detection replaces name, kind,
// storage and path when it recognises the folder.
export function newLibrary(number) {
  return {
    id: `new-library-${number}`,
    name: "New library",
    kind: "books",
    page_streaming: false,
    preferred_formats: defaultPreferredFormats("books"),
    source: {
      type: "filesystem",
      path: "",
    },
    inference: { folder_layout: "auto", filename_author: false },
  };
}
