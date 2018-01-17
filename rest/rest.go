package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/midbel/symon"
)

func Version() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		v := struct {
			Type    string        `json:"kernel"`
			Release string        `json:"release"`
			Uptime  time.Time     `json:"uptime"`
			Elapsed time.Duration `json:"duration"`
		}{}
		var err error
		if v.Type, v.Release, err = symon.Version(); err != nil {
			return nil, err
		}
		v.Uptime, v.Elapsed = symon.Uptime()
		return v, nil
	}
	return negociate(f)
}

func Process() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Process()
	}
	return negociate(f)
}

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
