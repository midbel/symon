package symon

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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

const (
	lastFile = "/var/log/lastlog"
	utmpFile = "/var/run/utmp"
	wtmpFile = "/var/log/wtmp"
)

var Epoch = time.Unix(0, 0)

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

//An user record as found in /var/log/lastlog
type Last struct {
	When time.Time `json:"timestamp"`
	Uid  int       `json:"uid"`
	Line string    `json:"line"`
	Host []byte    `json:"-"`
}

func (l Last) Found() bool {
	return !l.When.IsZero()
}

func (l Last) User() string {
	u, err := user.LookupId(strconv.Itoa(l.Uid))
	if err != nil {
		return fmt.Sprint(l.Uid)
	}
	return u.Username
}

//An user record as found in utmp and wtmp files
type Login struct {
	Record  uint32
	Pid     uint32
	Device  string
	Id      string
	User    string
	Host    string
	Seconds uint32
}

//MarshalJSON Implements the json.Marshaler MarshalJSON method.
func (l Login) MarshalJSON() ([]byte, error) {
	v := struct {
		Record string    `json:"record"`
		Pid    uint32    `json:"pid"`
		User   string    `json:"user"`
		Host   string    `json:"host"`
		When   time.Time `json:"dtstamp"`
	}{
		Record: l.Type(),
		Pid:    l.Pid,
		User:   l.User,
		Host:   l.Hostname(),
		When:   l.Since(),
	}
	return json.Marshal(v)
}

func (l Login) Hostname() string {
	if l.Remote() {
		return l.Host
	}
	return Hostname
}

func (l Login) Remote() bool {
	return l.Host != ""
}

func (l Login) Command() string {
	return processName(strconv.Itoa(int(l.Pid)), false)
}

func (l Login) Since() time.Time {
	return time.Unix(int64(l.Seconds), 0)
}

func (l Login) Type() string {
	if int(l.Record) >= len(records) {
		return "***"
	}
	return records[l.Record]
}

func Logins() (int, int) {
	var c, a int

	ds := []struct {
		Count *int
		File  string
	}{
		{File: utmpFile, Count: &c},
		{File: wtmpFile, Count: &a},
	}
	for _, d := range ds {
		i, err := os.Stat(d.File)
		if err != nil {
			return 0, 0
		}
		*d.Count = int(i.Size() / utmpRecordSize)
	}
	return c, a
}

//Last gives the users currently logged in on a system.
func Lastlog() ([]Last, error) {
	f, err := os.Open(lastFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		ls []Last
		wg sync.WaitGroup
	)
	for i, r := 0, bufio.NewReader(f); ; i++ {
		w := new(bytes.Buffer)
		if _, err := io.CopyN(w, r, int64(lastRecordSize)); err != nil {
			break
		}
		wg.Add(1)
		go func(n int, r *bytes.Buffer) {
			l, err := lastlog(n, r)
			if err == nil {
				ls = append(ls, *l)
			}
			wg.Done()
		}(i, w)
	}
	wg.Wait()
	return ls, nil
}

//Wtmp gives the full list from the startup of a system of users logging. It uses
///var/log/wtmp.
func Wtmp() ([]Login, error) {
	return scanWho(wtmpFile)
}

//Wtmp gives the full list from the startup of a system of users logging. It uses
///var/run/utmp.
func Utmp() ([]Login, error) {
	return scanWho(utmpFile)
}

func scanWho(path string) ([]Login, error) {
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

	var us []Login
	for s.Scan() {
		var u Login
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

func lastlog(i int, r *bytes.Buffer) (*Last, error) {
	if _, err := user.LookupId(strconv.Itoa(i)); err != nil {
		return nil, err
	}

	var s uint32
	binary.Read(r, binary.LittleEndian, &s)

	l := &Last{
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
