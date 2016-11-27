package symon

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

const proc = "/proc"

type Monitor struct {
	db     *bolt.DB
	ticker time.Ticker
	done   chan struct{}
}

func NewMonitor(db string, ttl int) (*Monitor, error) {
	if db, err := bolt.Open(db, 0644, nil); err != nil {
		return nil, err
	} else {
		m := &Monitor{
			db:     db,
			done:   make(chan struct{}),
			ticker: time.NewTicker(time.Second * time.Duration(ttl)),
		}
		go m.run()
		return m, nil
	}
}

func (m *Monitor) Close() error {
	m.ticker.Stop()
	close(m.done)

	return m.db.Close()
}

func (m *Monitor) run() {
	for {
		select {
		case <-m.done:
			return
		case <-m.ticker.C:
		}
	}
}

type Usage struct {
	Host        string        `json:"hostname"`
	Now         time.Time     `json:"timestamp"`
	Seconds     time.Duration `json:"uptime"`
	Users       []U           `json:"users"`
	Processes   []P           `json:"processes"`
	Memories    []M           `json:"memories"`
	Filesystems []F           `json:"filesystems"`
	Connections []C           `json:"connections"`
	Routes      []R           `json:"routes"`
	Err         error         `json:"error"`
	History     []Usage       `json:"history,omitempty"`
}

func Update(s time.Duration, done <-chan struct{}) <-chan Usage {
	ch := make(chan Usage)
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}
	go func() {
		ticker := time.NewTicker(s)
		defer func() {
			ticker.Stop()
			close(ch)
		}()
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				_, uptime := Uptime()
				u := Usage{Host: host, Now: t, Seconds: uptime}

				u.Users, u.Err = Utmp()
				u.Processes, u.Err = Processes()
				u.Memories, u.Err = Free()
				u.Filesystems, u.Err = Mount()
				u.Connections, u.Err = Netstat()
				u.Routes, u.Err = Route()

				ch <- u
			}
		}
	}()
	return ch
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
