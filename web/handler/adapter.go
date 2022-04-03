package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// ErrDone specifies that handler sent the response
// and that wrapper should not process returned value
// from the hanlder.
var ErrDone = errors.New("done")
var ErrNotFound = errors.New("not found")

// NoResponse is the special type that means
// that there will be no response from the handler.
type NoResponse *struct{}

func Wrap[T any](handler func(w http.ResponseWriter, r *http.Request) (T, error)) http.HandlerFunc {
	var empty T
	_, noResponse := any(empty).(NoResponse)

	return func(w http.ResponseWriter, r *http.Request) {
		data, err := handler(w, r)
		if err != nil {
			switch {
			case errors.Is(err, ErrDone):
				return
			case errors.Is(err, ErrNotFound):
				http.NotFound(w, r)
				return
			}

			log.Printf("handler failed: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !noResponse {
			w.Header().Set("Content-Type", "application/json")
		}

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("failed to encode response: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
