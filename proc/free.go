package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemInfo struct {
	Total   int64
	Free    int64
	Shared  int64
	Cached  int64
	Buffers int64
}

func (m MemInfo) Used() int64 {
	return m.Total - m.Free - m.Buffers - m.Cached
}

func Free() (MemInfo, MemInfo, error) {
	var (
		mem  MemInfo
		swap MemInfo
	)
	r, err := os.Open(memFile)
	if err != nil {
		return mem, swap, err
	}
	defer r.Close()

	scan := bufio.NewScanner(r)
	for scan.Scan() {
		field, value, ok := strings.Cut(scan.Text(), ":")
		if !ok {
			return mem, swap, fmt.Errorf("missing : in line")
		}
		value, _, _ = strings.Cut(strings.TrimSpace(value), " ")

		val, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return mem, swap, err
		}
		switch strings.ToLower(field) {
		case "memtotal":
			mem.Total = val
		case "memfree":
			mem.Free = val
		case "swaptotal":
			swap.Total = val
		case "swapfree":
			swap.Free = val
		case "shmem":
			mem.Shared = val
		case "buffers":
			mem.Buffers = val
		case "cached", "srreclaimable":
			mem.Cached += val
		default:
		}
	}
	return mem, swap, scan.Err()
}
