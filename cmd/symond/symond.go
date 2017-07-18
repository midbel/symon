package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/midbel/symon"
)

func main() {
	flag.Parse()

	http.HandleFunc("/users/", Users)
	http.HandleFunc("/process/", Process)
	if err := http.ListenAndServe(flag.Arg(0), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Process(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ps, err := symon.Processes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ps)
}

func Users(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var (
		who interface{}
		err error
	)
	switch _, base := path.Split(r.URL.Path); base {
	case "", "utmp":
		who, err = symon.Utmp()
	case "wtmp":
		who, err = symon.Wtmp()
	case "lastlog":
		who, err = symon.Last()
	case "faillog":
		w.WriteHeader(http.StatusNotImplemented)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(who)
}
