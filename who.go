package symon

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	lastRecordSize = 292
	utmpRecordSize = 384
)

var Epoch = time.Unix(0, 0)

var hostname string

var records = []string{
	"empty",
	"run",
	"boot",
	"new",
	"old",
	"init",
	"login",
	"user",
	"dead",
}

func init() {
	if h, err := os.Hostname(); err == nil {
		hostname = h
	} else {
		hostname = "localhost"
	}
}

//An user record as found in /var/log/lastlog
type L struct {
	When time.Time `json:"timestamp"`
	Uid  int       `json:"uid"`
	Line string    `json:"line"`
	Host []byte    `json:"-"`
}

func (l L) Found() bool {
	return !l.When.IsZero()
}

func (l L) User() string {
	u, err := user.LookupId(strconv.Itoa(l.Uid))
	if err != nil {
		return fmt.Sprint(l.Uid)
	}
	return u.Username
}

//An user record as found in utmp and wtmp files
type U struct {
	Record  uint32
	Pid     uint32
	Device  string
	Id      string
	User    string
	Host    string
	Seconds uint32
}

//MarshalJSON Implements the json.Marshaler MarshalJSON method.
func (u U) MarshalJSON() ([]byte, error) {
	v := struct {
		Record string    `json:"record"`
		Pid    uint32    `json:"pid"`
		User   string    `json:"user"`
		Host   string    `json:"host"`
		When   time.Time `json:"dtstamp"`
	}{
		Record: u.Type(),
		Pid:    u.Pid,
		User:   u.User,
		Host:   u.Hostname(),
		When:   u.Since(),
	}
	return json.Marshal(v)
}

func (u U) Hostname() string {
	if u.Remote() {
		return u.Host
	}
	return hostname
}

func (u U) Remote() bool {
	return u.Host != ""
}

func (u U) Command() string {
	return processName(strconv.Itoa(int(u.Pid)), false)
}

func (u U) Since() time.Time {
	return time.Unix(int64(u.Seconds), 0)
}

func (u U) Type() string {
	if int(u.Record) >= len(records) {
		return "***"
	}
	return records[u.Record]
}

func Fail() error {
	f, err := os.Open("/var/log/faillog")
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

//Last gives the users currently logged in on a system.
func Last() ([]L, error) {
	f, err := os.Open("/var/log/lastlog")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(func(bs []byte, ateof bool) (int, []byte, error) {
		if len(bs) < lastRecordSize {
			return 0, nil, nil
		}
		vs := make([]byte, lastRecordSize)
		copy(vs, bs[:lastRecordSize])
		return lastRecordSize, vs, nil
	})

	var ls []L
	var wg sync.WaitGroup
	for i := 0; s.Scan(); i++ {
		wg.Add(1)
		go func(n int, bs []byte) {
			l, err := lastlog(n, bs)
			if err == nil {
				ls = append(ls, *l)
			}
			wg.Done()
		}(i, s.Bytes())
	}
	wg.Wait()
	return ls, nil
}

//Wtmp gives the full list from the startup of a system of users logging. It uses
///var/log/wtmp.
func Wtmp() ([]U, error) {
	return scan("/var/log/wtmp")
}

//Wtmp gives the full list from the startup of a system of users logging. It uses
///var/run/utmp.
func Utmp() ([]U, error) {
	return scan("/var/run/utmp")
}

func scan(path string) ([]U, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(func(bs []byte, ateof bool) (int, []byte, error) {
		if len(bs) < utmpRecordSize {
			return 0, nil, nil
		}
		return utmpRecordSize, bs[:utmpRecordSize], nil
	})

	var us []U
	for s.Scan() {
		var u U
		r := bytes.NewBuffer(s.Bytes())

		binary.Read(r, binary.LittleEndian, &u.Record)
		binary.Read(r, binary.LittleEndian, &u.Pid)

		u.Device, u.Id = clean(r.Next(32)), clean(r.Next(4))
		u.User, u.Host = clean(r.Next(32)), clean(r.Next(256))
		r.Next(8)

		binary.Read(r, binary.LittleEndian, &u.Seconds)

		us = append(us, u)
	}
	sort.Slice(us, func(i, j int) bool {
		return us[i].Pid < us[j].Pid
	})
	return us, nil
}

func lastlog(i int, bs []byte) (*L, error) {
	if _, err := user.LookupId(strconv.Itoa(i)); err != nil {
		return nil, err
	}
	r := bytes.NewBuffer(bs)

	var s uint32
	binary.Read(r, binary.LittleEndian, &s)

	l := &L{
		Uid:  i,
		Line: clean(r.Next(32)),
		Host: bytes.Trim(r.Next(256), "\x00"),
	}

	if s > 0 {
		l.When = time.Unix(int64(s), 0)
	}
	return l, nil
}

func clean(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}
	bs = bytes.Trim(bs, "\x00")
	return strings.TrimSpace(string(bs))
}
