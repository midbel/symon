package symon

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const proc = "/proc"

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

func processName(pid string, c bool) string {
	var (
		bs  []byte
		err error
	)
	if !c {
		bs, err = ioutil.ReadFile(filepath.Join(proc, pid, "cmdline"))
		if err == nil {
			bs = bytes.Map(func(r rune) rune {
				if r == 0 {
					return ' '
				}
				return r
			}, bs)
			return strings.TrimSpace(string(bs))
		}
	}
	bs, err = ioutil.ReadFile(filepath.Join(proc, pid, "comm"))
	if err == nil {
		return strings.TrimSpace(string(bs))
	}
	return ""
}
