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

	var mem, swap M
	for s := bufio.NewScanner(f); s.Scan(); {
		ps := strings.FieldsFunc(s.Text(), func(r rune) bool {
			return r == ':' || r == ' ' || r == '\t'
		})
		switch f, v := strings.ToLower(ps[0]), ps[1]; f {
		case "memtotal":
			mem.Device = "mem"
			mem.Total, _ = strconv.Atoi(v)
		case "swaptotal":
			swap.Device = "swap"
			swap.Total, _ = strconv.Atoi(v)
		case "memfree":
			mem.Free, _ = strconv.Atoi(v)
		case "swapfree":
			swap.Free, _ = strconv.Atoi(v)
		case "buffers":
			mem.Buffers, _ = strconv.Atoi(v)
		case "cached", "slab":
			mem.Cache, _ = strconv.Atoi(v)
		case "memavailable":
			mem.Available, _ = strconv.Atoi(v)
		case "shmem":
			mem.Share, _ = strconv.Atoi(v)
		}
	}
	return []M{mem, swap}, nil
}
