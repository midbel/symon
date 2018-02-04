package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/midbel/symon"
)

func Stat() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Stat()
	}
	return negociate(f)
}

func Mount() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Mount()
	}
	return negociate(f)
}

func Net() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		v := struct {
			Routes     []symon.Route     `json:"routes"`
			Interfaces []symon.Interface `json:"interfaces"`
			Sockets    []symon.Socket    `json:"sockets"`
		}{}
		if vs, err := symon.Routes(); err == nil {
			v.Routes = vs
		}
		if vs, err := symon.Netstat(); err == nil {
			v.Sockets = vs
		}
		if vs, err := symon.Interfaces(); err == nil {
			v.Interfaces = vs
		}
		return v, nil
	}
	return negociate(f)
}

func Version() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		u, _ := symon.Logins()
		v := struct {
			Type    string    `json:"kernel"`
			Release string    `json:"release"`
			Uptime  time.Time `json:"uptime"`
			Users   int       `json:"users"`
			Process int       `json:"process"`
			Load    []float64 `json:"loadavg"`
		}{
			Type:    symon.Kernel,
			Release: symon.Distrib,
			Uptime:  symon.Boot,
			Process: len(symon.PIDs()),
			Load:    symon.Load(),
			Users:   u,
		}

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
