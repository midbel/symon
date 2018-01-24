package symon

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const proc = "/proc"

type S struct {
	Main    Core      `json:"cpu"`
	Cores   []Core    `json:"cores"`
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
	values := make([]float64, 3)
	for i := 0; i < len(values); i++ {
		value, err := r.ReadString(' ')
		if err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			values[i] = v
		}
	}
	return values
}

func Uptime() (time.Time, time.Duration) {
	f, err := os.Open(filepath.Join(proc, "uptime"))
	if err != nil {
		return time.Now(), time.Duration(0)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	value, err := r.ReadString(' ')
	if err != nil {
		return time.Now(), time.Duration(0)
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return time.Now(), time.Duration(0)
	}
	up := time.Duration(int64(secs)) * time.Second
	return time.Now().Add(-up), up
}

func Version() (string, string, error) {
	infos := make([]string, 2)
	for i, n := range []string{"ostype", "osrelease"} {
		bs, err := ioutil.ReadFile(filepath.Join(proc, "sys", "kernel", n))
		if err != nil {
			return "", "", err
		}
		infos[i] = strings.TrimSpace(string(bs))
	}
	bs, err := ioutil.ReadFile("/etc/issue.net")
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	return strings.Join(infos, " "), strings.TrimSpace(string(bs)), nil
}

type Core struct {
	Label  string  `json:"label"`
	User   float64 `json:"user"`
	Nice   float64 `json:"nice"`
	System float64 `json:"system"`
	Idle   float64 `json:"idle"`
	Wait   float64 `json:"iowait"`
}

func loadStatsCPU(v string, vs []string) Core {
	c := Core{Label: v}

	fs := make([]float64, len(vs))
	var n float64
	for i, v := range vs {
		fs[i], _ = strconv.ParseFloat(v, 64)
		n += fs[i]
	}

	cs := []*float64{&c.User, &c.Nice, &c.System, &c.Idle, &c.Wait}
	for i := 0; i < len(cs); i++ {
		*(cs[i]) = fs[i] / n
	}
	return c
}
