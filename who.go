package symon

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"os"
	"os/user"
	"sort"
	"strconv"
	"time"
)

const (
	lastRecordSize = 292
	utmpRecordSize = 384
)

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

type L struct {
	When time.Time `json:"timestamp"`
	User string    `json:"username"`
	Uid  int       `json:"uid"`
	Line string    `json:"line"`
	Host []byte    `json:"-"`
}

type U struct {
	Record  uint32 `json:"record"`
	Pid     uint32 `json:"pid"`
	Device  string `json:"device"`
	Id      string `json:"id"`
	User    string `json:"user"`
	Host    string `json:"host"`
	Seconds uint32 `json:"seconds"`
}

// func (u U) MarshalJSON() ([]byte, error) {
//
// }

func (u U) Hostname() string {
	if u.Remote() {
		return u.Host
	}
	h, _ := os.Hostname()
	return h
}

func (u U) Remote() bool {
	return u.Host != ""
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
		return lastRecordSize, bs[:lastRecordSize], nil
	})

	var data []L
	for i := 0; s.Scan(); i++ {
		if err := s.Err(); err != nil {
			return nil, err
		}
		r := bytes.NewBuffer(s.Bytes())
		var secs uint32
		if err := binary.Read(r, binary.LittleEndian, &secs); err != nil {
			return nil, err
		}
		if secs == 0 {
			continue
		}
		u, err := user.LookupId(strconv.Itoa(i))
		if err != nil {
			return nil, err
		}
		l := L{
			When: time.Unix(int64(secs), 0),
			User: u.Username,
			Uid:  i,
			Line: clean(r.Next(32)),
			Host: r.Next(256),
		}
		data = append(data, l)
	}
	return data, nil
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

func clean(buf []byte) string {
	buf = bytes.Trim(buf, "\x00")
	return string(buf)
}
