package catalog

// cloneBook copies a book so a derived snapshot cannot share backing storage
// with the scan that produced it.
func cloneBook(book Book) Book {
	book.Authors = append([]string(nil), book.Authors...)
	book.AuthorSort = append([]string(nil), book.AuthorSort...)
	book.Tags = append([]string(nil), book.Tags...)
	if book.Identifiers != nil {
		identifiers := make(map[string]string, len(book.Identifiers))
		for scheme, value := range book.Identifiers {
			identifiers[scheme] = value
		}
		book.Identifiers = identifiers
	}
	if book.SeriesIndex != nil {
		value := *book.SeriesIndex
		book.SeriesIndex = &value
	}
	return book
}

func cloneBooks(books []Book) []Book {
	out := make([]Book, len(books))
	for index := range books {
		out[index] = cloneBook(books[index])
	}
	return out
}

// booksEqual compares published catalogue values without relying on pointer
// identity for optional series indexes. It keeps no-op scans from invalidating
// every derived query index while still treating any user-visible or source
// revision change as a new generation. Every published field is compared: a
// Calibre sidecar edit can change tags, publisher, rating or identifiers while
// the book file keeps the same size and modification time.
func booksEqual(left, right []Book) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !bookEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

func bookEqual(left, right Book) bool {
	if left.ID != right.ID || left.Path != right.Path || left.Title != right.Title ||
		left.Series != right.Series || left.PublicationRole != right.PublicationRole ||
		left.Format != right.Format || left.MediaType != right.MediaType ||
		left.PageCount != right.PageCount || left.PageMediaType != right.PageMediaType ||
		left.Language != right.Language || left.Description != right.Description ||
		left.Identifier != right.Identifier || left.Size != right.Size ||
		left.Publisher != right.Publisher || left.Published != right.Published ||
		left.Rating != right.Rating || left.TitleSort != right.TitleSort ||
		left.ReadAlong != right.ReadAlong ||
		!left.Modified.Equal(right.Modified) || !left.CoverModified.Equal(right.CoverModified) ||
		left.MetadataRead != right.MetadataRead || !optionalFloatEqual(left.SeriesIndex, right.SeriesIndex) ||
		!stringsEqual(left.Authors, right.Authors) || !stringsEqual(left.AuthorSort, right.AuthorSort) ||
		!stringsEqual(left.Tags, right.Tags) || !identifiersEqual(left.Identifiers, right.Identifiers) {
		return false
	}
	return true
}

func optionalFloatEqual(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func identifiersEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for scheme, value := range left {
		if other, ok := right[scheme]; !ok || other != value {
			return false
		}
	}
	return true
}

func stringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
