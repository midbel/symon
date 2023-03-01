package proc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/midbel/shlex"
	"github.com/midbel/slices"
)

type ProcInfo struct {
	Pid      int
	Cmd      string
	Args     []string
	Status   rune
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
		if ifo.Args, err = readCmdline(filepath.Join(proc, f.Name())); err != nil {
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
	str = bytes.Trim(str, "\x00")
	str = bytes.Map(func(r rune) rune {
		if r == 0 {
			return ' '
		}
		return r
	}, str)
	return shlex.Split(bytes.NewReader(str))
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
			value = strings.TrimSpace(value)
			info.Status = slices.Fst([]rune(value))
		case "pid":
			pid, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return info, err
			}
			info.Pid = pid
		case "uid":
			uid, _, ok := strings.Cut(strings.TrimSpace(value), "\t")
			if !ok {
				break
			}
			u, err := user.LookupId(uid)
			if err != nil {
				return info, err
			}
			info.User = u.Username
		case "gid":
			gid, _, ok := strings.Cut(strings.TrimSpace(value), "\t")
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
