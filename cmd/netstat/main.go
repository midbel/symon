package main

import (
	"fmt"
	"os"

	"github.com/midbel/symon/proc"
)

func main() {
	conns, err := proc.Netstat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, c := range conns {
		fmt.Println(c.Proto, c.State, c.User, c.Local, c.Remote)
	}
}
