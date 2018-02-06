package symon

import (
	"bufio"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Load() []float64 {
	f, err := os.Open(filepath.Join(proc, "loadavg"))
	if err != nil {
		return nil
	}
	defer f.Close()

	r := bufio.NewReader(f)
	vs := make([]float64, 3)
	for i := 0; i < len(vs); i++ {
		t, err := r.ReadString(' ')
		if err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			vs[i] = v
		}
	}
	return vs
}

type Usage struct {
	Label string  `json:"name"`
	Total float64 `json:"total"`

	User   float64 `json:"user"`
	UserN  float64 `json:"usern"`
	Syst   float64 `json:"system"`
	Idle   float64 `json:"idle"`
	Wait   float64 `json:"iowait"`
	Hard   float64 `json:"hardirq"`
	Soft   float64 `json:"softirq"`
	Steal  float64 `json:"steal"`
	Guest  float64 `json:"guest"`
	GuestN float64 `json:"guestn"`
}

type times struct {
	Label  string
	Values []float64
}

func (t times) TotalTime() float64 {
	var v float64
	for i := range t.Values {
		v += t.Values[i]
	}
	return v
}

func (t times) IdleTime() float64 {
	return t.Values[3]
}

func (t times) Usage(p *times) *Usage {
	i := t.IdleTime() - p.IdleTime()
	d := t.TotalTime() - p.TotalTime()

	if d == 0 {
		return &Usage{Label: t.Label}
	}

	calc := func(ix int) float64 {
		v := (100 * (t.Values[ix] - p.Values[ix])) / d
		if v < 0 || math.IsNaN(v) {
			return 0
		}
		return v
	}
	g := 1000 * ((d - i) / d) / 10
	if math.IsNaN(g) || g < 0 {
		g = 0
	}

	return &Usage{
		Label:  t.Label,
		Total:  g,
		User:   calc(0),
		UserN:  calc(1),
		Syst:   calc(2),
		Idle:   calc(3),
		Wait:   calc(4),
		Hard:   calc(5),
		Soft:   calc(6),
		Steal:  calc(7),
		Guest:  calc(8),
		GuestN: calc(9),
	}
}

func Percents(e time.Duration) ([]*Usage, error) {
	f, err := os.Open(filepath.Join(proc, "stat"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	var cs []*times
	for i := 0; i < 2; i++ {
		for rs, err := r.ReadString('\n'); err == nil; rs, err = r.ReadString('\n') {
			if !strings.HasPrefix(rs, "cpu") {
				break
			}
			cs = append(cs, readCPUTimes(rs))
		}
		if i < 1 {
			time.Sleep(e)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		r.Reset(f)
	}
	us := make([]*Usage, len(cs)/2)
	for i, j := 0, len(us); i < j; i++ {
		us[i] = cs[i+j].Usage(cs[i])
	}
	return us, nil
}

func readCPUTimes(s string) *times {
	vs := strings.Fields(strings.TrimSpace(s))

	cs := make([]float64, len(vs)-1)
	for i, j := 1, 0; i < len(vs); i, j = i+1, j+1 {
		v, _ := strconv.ParseFloat(vs[i], 64)
		cs[j] = v / Tick
	}
	return &times{vs[0], cs}
}
