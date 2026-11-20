// The A–Z rail is only correct when its letters name offsets the server
// reported, and /api/book-navigation has a single (title) order. Under any other
// sort those offsets point at the wrong books, so the letters filter the shelf
// instead of jumping — the rail stays useful and lands on the letter asked for
// either way.
export function configureAlphabetRail(catalogAlphabet, {
  collectionMode,
  browseMode,
  sortedBooks,
  buckets,
  activeBucket,
  navigation,
}) {
  const jumping = !collectionMode && (browseMode !== "books" || !sortedBooks);
  catalogAlphabet.configure({
    jump: jumping,
    active: activeBucket,
    navigation: browseMode === "books" && jumping ? navigation() : buckets,
  });
}
