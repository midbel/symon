package symon

import (
	"bufio"
	"io"
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
	set := func(p *int, v string) {
		result, _ := strconv.Atoi(v)
		*p += result
	}
	f, err := os.Open(filepath.Join(proc, "meminfo"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	mem, swap := M{}, M{}
	for s.Scan() {
		if err := s.Err(); err != nil && err != io.EOF {
			return nil, err
		}
		parts := strings.Fields(s.Text())
		field, value := parts[0], strings.TrimSpace(parts[1])
		switch field := strings.ToLower(field[:len(field)-1]); field {
		case "memtotal":
			mem.Device = "mem"
			set(&mem.Total, value)
		case "swaptotal":
			swap.Device = "swap"
			set(&swap.Total, value)
		case "memfree":
			set(&mem.Free, value)
		case "swapfree":
			set(&swap.Free, value)
		case "buffers":
			set(&mem.Buffers, value)
		case "cached", "slab":
			set(&mem.Cache, value)
		case "memavailable":
			set(&mem.Available, value)
		case "shmem":
			set(&mem.Share, value)
		}
	}
	return []M{mem, swap}, nil
}
