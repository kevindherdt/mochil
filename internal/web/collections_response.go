package web

import (
	"errors"
	"net/http"
	"os"
)

var errWrongLibrary = errors.New("this collection belongs to another library")

// writeCollectionError turns a collection store error into the response that tells the caller what went wrong.
func writeCollectionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errSmartCollection):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, errWrongLibrary):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, os.ErrNotExist):
		http.NotFound(w, r)
	default:
		http.Error(w, "Collection could not be saved", http.StatusInternalServerError)
	}
}
