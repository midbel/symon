package symon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Size float64

const (
	Kilo Size = 1.0
	Mega Size = Kilo * 1024.0
	Giga Size = Mega * 1024.0
)

func (s *Size) String() string {
	return fmt.Sprint(*s)
}

func (s *Size) Set(v string) error {
	switch v {
	default:
		return fmt.Errorf("unknow unit %s", v)
	case "m":
		*s = Mega
	case "k", "":
		*s = Kilo
	case "g":
		*s = Giga
	}
	return nil
}

type M struct {
	Device    string
	Total     float64
	Free      float64
	Buffers   float64
	Cache     float64
	Available float64
	Share     float64
}

func (m M) Cumulate(o M) M {
	return M{
		Total:     m.Total + o.Total,
		Free:      m.Free + o.Free,
		Buffers:   m.Buffers + o.Buffers,
		Cache:     m.Cache + o.Cache,
		Share:     m.Share + o.Share,
		Available: m.Available + o.Available,
	}
}

func (m M) Scale(s Size) M {
	z := float64(s)
	return M{
		Device:    m.Device,
		Total:     m.Total / z,
		Free:      m.Free / z,
		Buffers:   m.Buffers / z,
		Cache:     m.Cache / z,
		Share:     m.Share / z,
		Available: m.Available / z,
	}
}

func (m M) MarshalJSON() ([]byte, error) {
	v := struct {
		Device string    `json:"device"`
		Total  float64   `json:"total"`
		Free   float64   `json:"free"`
		Used   float64   `json:"used"`
		When   time.Time `json:"dtstamp"`
	}{
		Device: m.Device,
		Total:  m.Total,
		Free:   m.Free,
		Used:   m.Used(),
		When:   time.Now(),
	}
	return json.Marshal(v)
}

func (m M) Used() float64 {
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

	return <-meminfo(f), nil
}

func meminfo(r io.ReadSeeker) <-chan []M {
	q := make(chan []M)
	go func() {
		defer close(q)
		for {
			var mem, swap M
			for s := bufio.NewScanner(r); s.Scan(); {
				ps := strings.FieldsFunc(s.Text(), func(r rune) bool {
					return r == ':' || r == ' ' || r == '\t'
				})
				switch f, v := strings.ToLower(ps[0]), ps[1]; f {
				case "memtotal":
					mem.Device = "mem"
					mem.Total, _ = strconv.ParseFloat(v, 64)
				case "swaptotal":
					swap.Device = "swap"
					swap.Total, _ = strconv.ParseFloat(v, 64)
				case "memfree":
					mem.Free, _ = strconv.ParseFloat(v, 64)
				case "swapfree":
					swap.Free, _ = strconv.ParseFloat(v, 64)
				case "buffers":
					mem.Buffers, _ = strconv.ParseFloat(v, 64)
				case "cached", "slab", "sreclaimable":
					n, _ := strconv.ParseFloat(v, 64)
					mem.Cache += n
				case "memavailable":
					mem.Available, _ = strconv.ParseFloat(v, 64)
				case "shmem":
					mem.Share, _ = strconv.ParseFloat(v, 64)
				}
			}
			q <- []M{mem, swap}
			r.Seek(io.SeekStart, 0)
		}
	}()
	return q
}
