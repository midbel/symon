package symon

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Process struct {
	Name   string        `json:"process"`
	State  string        `json:"state"`
	Uid    int           `json:"uid"`
	Gid    int           `json:"gid"`
	Pid    int           `json:"pid"`
	Parent int           `json:"ppid"`
	Mem    float64       `json:"mem"`
	Core   float64       `json:"cpu"`
	Uptime time.Duration `json:"uptime"`
}

func (p Process) MarshalJSON() ([]byte, error) {
	v := struct {
		Name    string `json:"process"`
		State   string `json:"state"`
		User    string `json:"user"`
		Group   string `json:"group"`
		Pid     int    `json:"pid"`
		Parent  int    `json:"ppid"`
		Command string `json:"command"`
	}{
		Name:    p.Name,
		State:   p.State,
		Pid:     p.Pid,
		Parent:  p.Parent,
		User:    p.User(),
		Group:   p.Group(),
		Command: p.Command(),
	}
	return json.Marshal(v)
}

func (p Process) User() string {
	id := strconv.Itoa(p.Uid)
	if u, err := user.LookupId(id); err != nil {
		return "***"
	} else {
		return u.Username
	}
}

func (p Process) Group() string {
	id := strconv.Itoa(p.Uid)
	if g, err := user.LookupGroupId(id); err != nil {
		return "***"
	} else {
		return g.Name
	}
}

func (p Process) Device() string {
	if p, err := filepath.EvalSymlinks(filepath.Join(proc, strconv.Itoa(p.Pid), "fd", "0")); err != nil {
		return ""
	} else {
		parts := strings.Split(p, string(os.PathSeparator))
		if _, err := strconv.Atoi(parts[len(parts)-1]); err != nil {
			return ""
		} else {
			return filepath.Join(parts[2:]...)
		}
	}
}

func (p Process) Command() string {
	return processName(strconv.Itoa(p.Pid), true)
}

func PIDs() []int {
	is, err := ioutil.ReadDir(proc)
	if err != nil {
		return nil
	}
	ps := make([]int, 0, len(is))
	for _, i := range is {
		if !i.IsDir() {
			continue
		}
		if v, err := strconv.Atoi(i.Name()); err == nil {
			ps = append(ps, v)
		}
	}
	return ps
}

//Processes returns the list of process currently exectued on a system. It tries
//to copy the behavior of the `ps` command.
func Processes() ([]Process, error) {
	data := make([]Process, 0, 100)

	var wg sync.WaitGroup
	err := filepath.Walk(proc, func(path string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !i.IsDir() || path == proc {
			return nil
		}
		if _, err := strconv.Atoi(i.Name()); err != nil {
			return filepath.SkipDir
		}
		f, err := os.Open(filepath.Join(path, "status"))
		switch {
		case err == nil:
		case os.IsNotExist(err):
			return filepath.SkipDir
		default:
			return err
		}
		defer f.Close()

		var p Process
		for s := bufio.NewScanner(f); s.Scan(); {
			parts := strings.Split(s.Text(), ":")
			if len(parts) <= 1 {
				continue
			}
			switch field, value := strings.ToLower(parts[0]), strings.TrimSpace(parts[1]); field {
			case "name":
				p.Name = value
			case "state":
				p.State = value
			case "pid":
				p.Pid, _ = strconv.Atoi(value)
			case "ppid":
				p.Parent, _ = strconv.Atoi(value)
			case "uid":
				parts := strings.Fields(value)
				p.Uid, _ = strconv.Atoi(parts[0])
			case "gid":
				parts := strings.Fields(value)
				p.Gid, _ = strconv.Atoi(parts[0])
			}
		}
		wg.Add(1)
		go func(v *Process) {
			v.Core, v.Uptime = readProcessStats(v.Pid, 5, time.Millisecond*10)
			data = append(data, *v)
			wg.Done()
		}(&p)
		return nil
	})
	wg.Wait()
	return data, err
}

func readProcessStats(p, c int, e time.Duration) (float64, time.Duration) {
	t := time.NewTicker(e)
	defer t.Stop()

	var (
		pu, ps, ct float64
		pt         time.Time
		up         time.Duration
	)
	for i := 0; i < c; i++ {
		bs, err := ioutil.ReadFile(filepath.Join(proc, strconv.Itoa(p), "stat"))
		if err != nil {
			return 0, 0
		}
		fs := strings.Fields(string(bs))
		u, _ := strconv.ParseFloat(fs[13], 64)
		s, _ := strconv.ParseFloat(fs[14], 64)

		w := <-t.C
		u, s = u/Tick, s/Tick
		dv, dt := (u-pu)+(s-ps), w.Sub(pt)

		ct += 100 * (dv / dt.Seconds())
		pu, ps, pt = u, s, w

		j, _ := strconv.ParseFloat(fs[21], 64)
		up = time.Duration(j/Tick) * time.Second
	}
	return ct / float64(c), time.Since(Boot.Add(up))
}
