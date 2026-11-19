package web

// pageWindow turns a request-supplied offset and limit into a slice window.
// The offset comes straight from a query string and can be as large as MaxInt,
// so the end is computed as start + min(limit, total-start): adding offset and
// limit first overflows to a negative number and the slice expression then
// panics instead of answering an empty page.
func pageWindow(offset, limit, total int) (start, end int) {
	total = max(total, 0)
	start = min(max(offset, 0), total)
	end = start + min(max(limit, 0), total-start)
	return start, end
}
