package symon

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type psort []P

func (p psort) Len() int { return len(p) }

func (p psort) Less(i, j int) bool { return p[i].Pid < p[j].Pid }

func (p psort) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type P struct {
	Name   string `json:"process"`
	State  string `json:"state"`
	Uid    int    `json:"uid"`
	Gid    int    `json:"gid"`
	Pid    int    `json:"pid"`
	Parent int    `json:"ppid"`
}

func (p P) User() string {
	id := strconv.Itoa(p.Uid)
	if u, err := user.LookupId(id); err != nil {
		return "***"
	} else {
		return u.Username
	}
}

func (p P) Group() string {
	id := strconv.Itoa(p.Uid)
	if g, err := user.LookupGroupId(id); err != nil {
		return "***"
	} else {
		return g.Name
	}
}

func (p P) Device() string {
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

func (p P) Command() string {
	path := filepath.Join(proc, strconv.Itoa(p.Pid), "cmdline")
	if buf, err := ioutil.ReadFile(path); err == nil {
		parts := make([]string, 0, 12)
		for {
			if ix := bytes.IndexByte(buf, 0x0); ix >= 0 {
				str := strings.TrimSpace(string(buf[:ix]))
				if len(str) > 0 {
					parts = append(parts, str)
				}
				buf = buf[ix+1:]
			} else {
				break
			}
		}
		if len(parts) == 0 || len(parts) >= 5 {
			return p.Name
		}
		return strings.Join(parts, " ")
	}
	return p.Name
}

//Processes returns the list of process currently exectued on a system. It tries
//to copy the behavior of the `ps` command.
func Processes() ([]P, error) {
	data := make([]P, 0, 100)
	err := filepath.Walk(proc, func(path string, i os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == proc || !i.IsDir() {
			return nil
		}
		if _, err := strconv.Atoi(i.Name()); err != nil {
			return filepath.SkipDir
		}
		f, err := os.Open(filepath.Join(path, "status"))
		if err != nil {
			return err
		}
		defer f.Close()
		p := P{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil && err != io.EOF {
				return err
			}
			parts := strings.Split(scanner.Text(), ":")
			if len(parts) <= 1 {
				continue
			}
			field, value := parts[0], strings.TrimSpace(parts[1])
			switch field := strings.ToLower(field); field {
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
		data = append(data, p)
		return nil
	})
	sort.Sort(psort(data))
	return data, err
}
