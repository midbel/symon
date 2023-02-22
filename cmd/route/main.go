package main

import (
	"fmt"
	"os"

	"github.com/midbel/symon/proc"
)

func main() {
	routes, err := proc.Routes()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, r := range routes {
		fmt.Println(r.Interface, r.Network, r.Gateway)
	}
}
