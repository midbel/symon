package symon

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var Tick float64 = 100.0

func init() {
	if t, err := strconv.ParseFloat(os.Getenv("CLK_TCK"), 64); err == nil {
		Tick = t
	}
}

type S struct {
	Main    *Core     `json:"cpu"`
	Cores   []*Core   `json:"cores"`
	Boot    time.Time `json:"boot"`
	Forks   int64     `json:"forks"`
	Running int64     `json:"running"`
	Waiting int64     `json:"waiting"`
}

func Stat() (*S, error) {
	f, err := os.Open(filepath.Join(proc, "stat"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat := new(S)
	for s := bufio.NewScanner(f); s.Scan(); {
		fs := strings.SplitN(s.Text(), " ", 2)
		switch v, vs := fs[0], strings.Fields(fs[1]); {
		case strings.HasPrefix(v, "cpu") && v != "cpu":
			stat.Cores = append(stat.Cores, loadStatsCPU(v, vs))
		case v == "cpu":
			stat.Main = loadStatsCPU(v, vs)
		case v == "btime":
			t, _ := strconv.ParseInt(vs[0], 10, 64)
			stat.Boot = time.Unix(t, 0)
		case v == "processes":
			stat.Forks, _ = strconv.ParseInt(vs[0], 10, 64)
		case v == "procs_running":
			stat.Running, _ = strconv.ParseInt(vs[0], 10, 64)
		case v == "procs_blocked":
			stat.Waiting, _ = strconv.ParseInt(vs[0], 10, 64)
		}
	}
	return stat, nil
}

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

func TotalPercentCPU(e time.Duration) <-chan float64 {
	q := make(chan float64)
	go func() {
		defer close(q)
		f, err := os.Open(filepath.Join(proc, "stat"))
		if err != nil {
			return
		}
		defer f.Close()
		r := bufio.NewReader(f)

		var total, idle float64
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				return
			}
			vs := strings.SplitN(s, " ", 2)
			if vs[0] != "cpu" {
				return
			}
			c := loadStatsCPU(vs[0], strings.Fields(vs[1]))
			i, t := c.IdleTime(), c.TotalTime()
			idle, total = i-idle, t-total

			<-time.After(e)
			f.Seek(io.SeekStart, 0)
			r.Reset(f)
			q <- (1000 * (total - idle) / total) / 10

			idle, total = i, t
		}
	}()
	return q
}

type Core struct {
	Label  string  `json:"label"`
	User   float64 `json:"user"`
	UserN  float64 `json:"usern"`
	Syst   float64 `json:"system"`
	Idle   float64 `json:"idle"`
	Wait   float64 `json:"iowait"`
	Irq    float64 `json:"irq"`
	Soft   float64 `json:"softirq"`
	Steal  float64 `json:"steal"`
	Guest  float64 `json:"guest"`
	GuestN float64 `json:"guestn"`
}

// func (c Core) Diff(p Core) (*Core, error) {
// 	return nil, nil
// }

func (c Core) BusyTime() float64 {
	return c.TotalTime() - c.IdleTime()
}

func (c Core) TotalTime() float64 {
	n := c.User + c.UserN + c.Syst + c.Syst + c.Idle + c.Wait + c.Irq + c.Soft + c.Steal
	return n - c.Guest - c.GuestN
}

func (c Core) IdleTime() float64 {
	return c.Idle
}

func loadStatsCPU(v string, vs []string) *Core {
	c := &Core{Label: v}

	cs := []*float64{
		&c.User,
		&c.UserN,
		&c.Syst,
		&c.Idle,
		&c.Wait,
		&c.Irq,
		&c.Soft,
		&c.Steal,
		&c.Guest,
		&c.GuestN,
	}
	for i := 0; i < len(vs); i++ {
		v, _ := strconv.ParseFloat(vs[i], 64)
		*(cs[i]) = v / Tick
	}
	return c
}
