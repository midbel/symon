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

type usort []U

func (u usort) Len() int { return len(u) }

func (u usort) Less(i, j int) bool { return u[i].Pid < u[j].Pid }

func (u usort) Swap(i, j int) { u[i], u[j] = u[j], u[i] }

type L struct {
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"username"`
	Uid       int       `json:"uid"`
	Line      string    `json:"line"`
	Host      []byte    `json:"-"`
}

type U struct {
	Record  int    `json:"record"`
	Pid     int    `json:"pid"`
	Device  string `json:"device"`
	Id      string `json:"id"`
	User    string `json:"user"`
	Host    string `json:"host"`
	Seconds int    `json:"seconds"`
}

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
	if u.Record >= len(records) {
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
	const size = 292

	f, err := os.Open("/var/log/lastlog")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(func(data []byte, ateof bool) (int, []byte, error) {
		if len(data) < size {
			return 0, nil, nil
		}
		return size, data[:size], nil
	})

	var data []L
	if stat, err := f.Stat(); err == nil {
		data = make([]L, 0, int(stat.Size())/size)
	} else {
		data = make([]L, 0, 32)
	}

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
			Timestamp: time.Unix(int64(secs), 0),
			User:      u.Username,
			Uid:       i,
			Line:      clean(r.Next(32)),
			Host:      r.Next(256),
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
	const size = 384

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(func(data []byte, ateof bool) (int, []byte, error) {
		if len(data) < size {
			return 0, nil, nil
		}
		return size, data[:size], nil
	})

	s, _ := f.Stat()
	data := make([]U, 0, int(s.Size())/size)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		buf := scanner.Bytes()
		u := U{
			Record:  int(binary.LittleEndian.Uint32(buf[:4])),
			Pid:     int(binary.LittleEndian.Uint16(buf[4:8])),
			Device:  clean(buf[8:40]),
			Id:      clean(buf[40:44]),
			User:    clean(buf[44:76]),
			Host:    clean(buf[76:332]),
			Seconds: int(binary.LittleEndian.Uint32(buf[340:344])),
		}

		data = append(data, u)
	}
	sort.Sort(usort(data))
	return data, nil
}

func clean(buf []byte) string {
	buf = bytes.Trim(buf, "\x00")
	return string(buf)
}
