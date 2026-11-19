package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// Budgets for how long a client may take to send a request body. The byte limits cap size; these cap patience, so a
// connection that sends headers and then trickles a body cannot hold a handler open. Long media responses are unaffected.
var (
	smallBodyBudget  = 20 * time.Second
	uploadBodyBudget = 5 * time.Minute
)

// limitRequestBody bounds both the size of the body and the time allowed to read it.
func limitRequestBody(w http.ResponseWriter, r *http.Request, maximumBytes int64, budget time.Duration) {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBytes)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(budget))
}

// decodeJSONRequest applies a transport limit and accepts exactly one JSON
// value. Feature handlers retain ownership of shape and semantic validation.
func decodeJSONRequest(w http.ResponseWriter, r *http.Request, maximumBytes int64, target any, disallowUnknown bool) error {
	limitRequestBody(w, r, maximumBytes, smallBodyBudget)
	decoder := json.NewDecoder(r.Body)
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request contains multiple JSON values")
		}
		return err
	}
	return nil
}
