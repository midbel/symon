package main

import (
	"fmt"
	"os"

	"github.com/midbel/symon/proc"
)

func main() {
	list, err := proc.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, i := range list {
		fmt.Printf("%+v\n", i)
	}
}
