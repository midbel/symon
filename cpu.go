package symon

import (
	"bufio"
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

type Jiffy struct {
	Label string `json:"name"`
	When  time.Time `json:"timestamp"`

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

func (j Jiffy) TotalTime() float64 {
	return j.User + j.UserN + j.Syst + j.Idle + j.Wait + j.Hard + j.Soft + j.Steal
}

func (j Jiffy) IdleTime() float64 {
	return j.Idle
}

func (j Jiffy) Usage(p Jiffy) *Usage {
	i := j.IdleTime() - p.IdleTime()
	d := j.TotalTime() - p.TotalTime()

	if d == 0 {
		return &Usage{Label: j.Label}
	}

	calc := func(c, p float64) float64 {
		v := (100 * (c - p)) / d
		if v < 0 || math.IsNaN(v) {
			return 0
		}
		return v
	}

	return &Usage{
		Label:  j.Label,
		When:  j.When,
		Total:  calc(d, i),
		User:   calc(j.User, p.User),
		UserN:  calc(j.UserN, p.UserN),
		Syst:   calc(j.Syst, p.Syst),
		Idle:   calc(j.Idle, p.Idle),
		Wait:   calc(j.Wait, p.Wait),
		Hard:   calc(j.Hard, p.Hard),
		Soft:   calc(j.Soft, p.Soft),
		Steal:  calc(j.Steal, p.Steal),
		Guest:  calc(j.Guest, p.Guest),
		GuestN: calc(j.GuestN, p.GuestN),
	}
}

func Times() ([]Jiffy, error) {
	qs, err := readProcFile(filepath.Join(proc, "stat"), 11, 0, ' ')
	if err != nil {
		return nil, err
	}
	var js []Jiffy
	for rs := range qs {
		j := Jiffy{Label: rs[0], When: time.Now()}
		ts := []*float64{
			&j.User,
			&j.UserN,
			&j.Syst,
			&j.Idle,
			&j.Wait,
			&j.Hard,
			&j.Soft,
			&j.Steal,
			&j.Guest,
			&j.GuestN,
		}
		for i, j := 1, 0; i < len(ts); i, j = i+1, i {
			v, err := strconv.ParseFloat(rs[i], 64)
			if err != nil {
				return nil, err
			}
			*(ts[j]) = v / Tick
		}
		js = append(js, j)
	}
	return js, nil
}

type Usage struct {
	Label string  `json:"name"`
	Total float64 `json:"total"`
	When time.Time `json:"dtstamp"`

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

func Percents(e time.Duration) ([]*Usage, error) {
	var js []Jiffy
	for i := 0; i < 2; i++ {
		vs, err := Times()
		if err != nil {
			return nil, err
		}
		js = append(js, vs...)
		if i < 1 {
			time.Sleep(e)
		}
	}
	us := make([]*Usage, len(js)/2)
	for i, j := 0, len(us); i < j; i++ {
		us[i] = js[i+j].Usage(js[i])
	}
	return us, nil
}
