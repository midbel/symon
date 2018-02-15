package rest

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/midbel/symon"
)

func Mount() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Mount()
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

func Processes() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Processes()
	}
	return negociate(f)
}

func Interfaces() http.Handler {
	type ifi struct {
		When time.Time `json:"dtstamp"`
		symon.Interface
	}
	f := func(r *http.Request) (interface{}, error) {
		is, err := symon.Interfaces()
		if err != nil {
			return nil, err
		}
		vs := make([]ifi, len(is))
		n := time.Now()
		for j, i := range is {
			vs[j] = ifi{n, i}
		}
		return vs, nil
	}
	return negociate(f)
}

func Stats() http.Handler {
	var (
		ps []symon.Jiffy
		us []symon.Usage
		mu sync.RWMutex
	)
	go func() {
		ts := time.Tick(time.Second)
		for range ts {
			js, err := symon.Times()
			if err != nil {
				continue
			}
			if len(ps) > 0 {
				mu.Lock()
				vs := make([]symon.Usage, len(js))
				for i := 0; i < len(js); i++ {
					vs[i] = *(js[i].Usage(ps[i]))
				}
				us = vs
				mu.Unlock()
			}
			ps = js
		}
	}()
	f := func(r *http.Request) (interface{}, error) {
		mu.RLock()
		defer mu.RUnlock()
		v := struct {
			Times  []symon.Jiffy `json:"times"`
			Usages []symon.Usage `json:"usages,omitempty"`
		}{
			Times:  ps,
			Usages: us,
		}
		return v, nil
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
