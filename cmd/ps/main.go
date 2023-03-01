package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/midbel/symon/proc"
)

func main() {
	list, err := proc.Process()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Pid < list[j].Pid
	})
	for _, i := range list {
		fmt.Printf("%-8d %-16s %-16s %-4c %s", i.Pid, i.User, i.Group, i.Status, i.Cmd)
		fmt.Println()
	}
}
