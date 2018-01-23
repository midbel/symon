package rest

import (
	"encoding/json"
	// "net"
	"net/http"
	"time"

	"github.com/midbel/symon"
)

func Mount() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Mount()
	}
	return negociate(f)
}

func Routes() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		return symon.Routes()
	}
	return negociate(f)
}

func Netstat() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		q := r.URL.Query()
		return symon.Netstat(q["protocol"]...)
		// if err != nil {
		// 	return nil, err
		// }
		// if q.Get("resolve") != "" {
		// 	var h string
		// 	for i, s := range ns {
		// 		h, _, _ = net.SplitHostPort(s.Local)
		// 		if vs, err := net.LookupAddr(h); err == nil && len(vs) > 0 {
		// 			s.Local = vs[0]
		// 		}
		// 		h, _, _ = net.SplitHostPort(s.Remote)
		// 		if vs, err := net.LookupAddr(h); err == nil && len(vs) > 0 {
		// 			s.Local = vs[0]
		// 		}
		// 		ns[i] = s
		// 	}
		// }
		// return ns, err
	}
	return negociate(f)
}

func Version() http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		v := struct {
			Type    string        `json:"kernel"`
			Release string        `json:"release"`
			Uptime  time.Time     `json:"uptime"`
			Elapsed time.Duration `json:"duration"`
			Users   int           `json:"users"`
			Process int           `json:"process"`
			Load    []float64     `json:"loadavg"`
		}{}
		var err error
		if v.Type, v.Release, err = symon.Version(); err != nil {
			return nil, err
		}
		if us, err := symon.Utmp(); err == nil {
			v.Users = len(us)
		}
		if ps, err := symon.Process(); err == nil {
			v.Process = len(ps)
		}
		v.Uptime, v.Elapsed = symon.Uptime()

		if ps, err := symon.Process(); err == nil {
			v.Process = len(ps)
		}
		if us, err := symon.Utmp(); err == nil {
			v.Users = len(us)
		}
		v.Load = symon.Load()

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
