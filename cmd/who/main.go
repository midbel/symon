package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/midbel/symon/proc"
)

func main() {
	var (
		all  = flag.Bool("a", false, "get all user(s) currently logged in")
		sys  = flag.Bool("s", false, "include system user(s)")
		list []proc.Who
		err  error
	)
	flag.Parse()
	if *all {
		list, err = proc.All()
	} else {
		list, err = proc.Current()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].When.Before(list[j].When)
	})

	fmt.Printf("%-12s %-16s %-8s %-8s %s", "user", "from", "tty", "login", "command")
	fmt.Println()
	for _, i := range list {
		if !*sys && !i.Regular() {
			continue
		}
		fmt.Printf("%-12s %-16s %-8s %-8s %s", i.User, i.Addr, i.Terminal, i.When.Format("15:04"), i.Command())
		fmt.Println()
	}
}
