package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	var (
		addr  = flag.String("a", ":8080", "listening address")
		delay = flag.Duration("d", time.Second, "update interval")
	)
	flag.Parse()

	mon := Monitor()
	go mon.Run(*delay)

	http.Handle("/", handleStatus(mon))
	http.Handle("/process", handleProcess(mon))
	http.Handle("/memory", handleFree(mon))
	http.Handle("/loadavg", handleLoadAvg(mon))
	http.Handle("/users", handleUsers(mon))
	http.Handle("/netstat", handleNetstat(mon))

	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
