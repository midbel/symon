package main

import (
	"fmt"
	"os"

	"github.com/midbel/symon/proc"
)

func main() {
	mem, swap, err := proc.Free()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(mem)
	fmt.Println(swap)
}
