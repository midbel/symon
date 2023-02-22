package proc

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcInfo struct {
	Pid      int
	Cmd      string
	Args     []string
	Status   string
	User     string
	Group    string
	Nice     int
	Priority int
}

func Process() ([]ProcInfo, error) {
	files, err := os.ReadDir(proc)
	if err != nil {
		return nil, err
	}
	var list []ProcInfo
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(f.Name()); err != nil {
			continue
		}
		ifo, err := readProcInfo(filepath.Join(proc, f.Name()))
		if err != nil {
			return nil, err
		}
		list = append(list, ifo)
	}
	return list, nil
}

func readCmdline(dir string) ([]string, error) {
	str, err := os.ReadFile(filepath.Join(dir, procCmdline))
	if err != nil {
		return nil, err
	}
	_ = str
	return nil, nil
}

func readProcInfo(dir string) (ProcInfo, error) {
	var info ProcInfo

	r, err := os.Open(filepath.Join(dir, procStatus))
	if err != nil {
		return info, err
	}
	defer r.Close()

	scan := bufio.NewScanner(r)
	for scan.Scan() {
		field, value, ok := strings.Cut(scan.Text(), ":")
		if !ok {
			return info, fmt.Errorf("missing : in line")
		}
		switch strings.ToLower(field) {
		case "name":
			info.Cmd = strings.TrimSpace(value)
		case "state":
		case "pid":
			pid, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return info, err
			}
			info.Pid = pid
		case "uid":
			uid, _, ok := strings.Cut(strings.TrimSpace(value), " ")
			if !ok {
				break
			}
			u, err := user.LookupId(uid)
			if err != nil {
				return info, err
			}
			info.User = u.Username
		case "gid":
			gid, _, ok := strings.Cut(strings.TrimSpace(value), " ")
			if !ok {
				break
			}
			g, err := user.LookupGroupId(gid)
			if err != nil {
				return info, err
			}
			info.Group = g.Name
		default:
		}
	}
	return info, scan.Err()
}
