package rest

import (
	"encoding/json"
	"net/http"

	"github.com/midbel/symon"
)

func Free() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Free()
	}
	return negociate(f)
}

func Who() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Utmp()
	}
	return negociate(f)
}

type handler func(*http.Request) (interface{}, error)

func negociate(h handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		d, err := h(r)

		w.Header().Set("content-type", "application/json")
		switch err {
		case nil:
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if d == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		json.NewEncoder(w).Encode(d)
	}
	return http.HandlerFunc(f)
}
