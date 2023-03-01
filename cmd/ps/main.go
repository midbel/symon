package main

import (
	"fmt"
	"os"

	"github.com/midbel/symon/proc"
)

func main() {
	list, err := proc.Process()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, i := range list {
		fmt.Println(i.Pid, i.Cmd, i.Args)
	}
}
