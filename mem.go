package symon

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type M struct {
	Device    string `json:"device"`
	Total     int    `json:"total"`
	Free      int    `json:"free"`
	Buffers   int    `json:"buffers"`
	Cache     int    `json:"cache"`
	Available int    `json:"available"`
	Share     int    `json:"share"`
}

func (m M) Used() int {
	return m.Total - m.Free - m.Buffers - m.Cache
}

//Free gives the memory used by a system in a slice. The first element is the
//RAM used, the second element is the swap usage.
func Free() ([]M, error) {
	f, err := os.Open(filepath.Join(proc, "meminfo"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	var mem, swap M
	for s.Scan() {
		parts := strings.Fields(s.Text())
		field, value := parts[0], strings.TrimSpace(parts[1])
		switch field := strings.ToLower(field[:len(field)-1]); field {
		case "memtotal":
			mem.Device = "mem"
			mem.Total, _ = strconv.Atoi(value)
		case "swaptotal":
			swap.Device = "swap"
			swap.Total, _ = strconv.Atoi(value)
		case "memfree":
			mem.Free, _ = strconv.Atoi(value)
		case "swapfree":
			swap.Free, _ = strconv.Atoi(value)
		case "buffers":
			mem.Buffers, _ = strconv.Atoi(value)
		case "cached", "slab":
			mem.Cache, _ = strconv.Atoi(value)
		case "memavailable":
			mem.Available, _ = strconv.Atoi(value)
		case "shmem":
			mem.Share, _ = strconv.Atoi(value)
		}
	}
	return []M{mem, swap}, nil
}
