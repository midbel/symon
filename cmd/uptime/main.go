package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/midbel/slices"
	"github.com/midbel/symon/proc"
)

func main() {
	var (
		since  = flag.Bool("s", false, "system up since")
		pretty = flag.Bool("p", false, "show uptime in pretty format")
	)
	flag.Parse()

	switch {
	default:
		var (
			up, _  = proc.Uptime()
			avg, _ = proc.LoadAvg()
		)
		fmt.Fprintf(os.Stdout, "up: %s, load average: %.2f, %.2f, %.2f", up, slices.Fst(avg), slices.Snd(avg), slices.Lst(avg))
		fmt.Fprintln(os.Stdout)
	case *pretty:
		el := prettyTime()
		fmt.Fprint(os.Stdout, "up")
		if el.Days > 0 {
			fmt.Fprintf(os.Stdout, " %d days,", el.Days)
		}
		if el.Hours > 0 {
			fmt.Fprintf(os.Stdout, " %d hours,", el.Hours)
		}
		fmt.Fprintf(os.Stdout, " %d minutes", el.Minutes)
		fmt.Fprintln(os.Stdout)
	case *since:
		when, _ := proc.BootTime()
		fmt.Fprintln(os.Stdout, when.Format("2006-01-02 15:04:05"))
	}
}

type Elapsed struct {
	Days    int
	Hours   int
	Minutes int
}

func prettyTime() Elapsed {
	var (
		el      Elapsed
		up, err = proc.Uptime()
	)
	if err != nil {
		return el
	}

	days := up / (time.Hour * 24)
	up -= days * time.Hour * 24

	hours := up / time.Hour
	up -= hours * time.Hour

	mins := up / time.Minute

	el.Days = int(days)
	el.Hours = int(hours)
	el.Minutes = int(mins)

	return el
}
