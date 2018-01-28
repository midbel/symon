package symon

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Layers []string

func (a *Layers) String() string {
	return fmt.Sprint(*a)
}

func (a *Layers) Set(v string) error {
	for _, v := range strings.Split(v, ",") {
		switch v := strings.ToLower(v); v {
		case "udp", "tcp", "unix", "tcp6", "udp6":
			*a = append(*a, v)
		default:
			return fmt.Errorf("unknown protocol %s", v)
		}
	}
	return nil
}

type service struct {
	port, proto, name string
	aliases           []string
}

var services []service

func init() {
	f, err := os.Open("/etc/services")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		t := s.Text()
		if len(t) == 0 || t[0] == '#' {
			continue
		}
		if ix := strings.Index(t, "#"); ix >= 0 {
			t = t[:ix]
		}
		fs := strings.Fields(t)
		ps := strings.Split(fs[1], "/")
		s := service{
			port:  ps[0],
			proto: ps[1],
			name:  fs[0],
		}
		services = append(services, s)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].port <= services[j].port
	})
}

var states = []string{
	"established",
	"syn_sent",
	"syn_recv",
	"fin_wait1",
	"fin_wait2",
	"time_wait",
	"close",
	"close_wait",
	"last_ack",
	"listen",
	"closing",
}

const (
	ESTABLISHED = iota + 1
	SYN_SENT
	SYN_RECV
	FIN_WAIT1
	FIN_WAIT2
	TIME_WAIT
	CLOSE
	CLOSE_WAIT
	LAST_ACK
	LISTEN
	CLOSING
)

type C struct {
	Proto   string `json:"protocol"`
	Local   string `json:"local"`
	Remote  string `json:"remote"`
	State   int    `json:"state"`
	Uid     int    `json:"uid"`
	Recv    int    `json:"recv"`
	Send    int    `json:"send"`
	Command string `json:"command"`
}

func (c C) MarshalJSON() ([]byte, error) {
	v := struct {
		Proto   string `json:"protocol"`
		Local   string `json:"local"`
		Remote  string `json:"remote"`
		State   string `json:"state"`
		Recv    int    `json:"recv"`
		Send    int    `json:"send"`
		Command string `json:"command"`
	}{
		Proto:   c.Proto,
		Local:   c.Local,
		Remote:  c.Remote,
		State:   c.Status(),
		Recv:    c.Recv,
		Send:    c.Send,
		Command: c.Command,
	}
	return json.Marshal(v)
}

func (c C) User() string {
	u, err := user.LookupId(strconv.Itoa(c.Uid))
	if err == nil {
		return u.Username
	}
	return ""
}

func (c C) Status() string {
	if ix := c.State - 1; ix > len(states) {
		return "-"
	} else {
		return states[ix]
	}
}

type R struct {
	Interface string `json:"interface"`
	Address   string `json:"address"`
	Gateway   string `json:"gateway"`
	Mask      string `json:"mask"`
	Metric    int    `json:"metric"`
	Distance  int    `json:"distance"`
}

//Route gives the list of network routes currently known by a system.
func Routes() ([]R, error) {
	f, err := os.Open(filepath.Join(proc, "net", "route"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan()

	data := make([]R, 0, 16)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}
		parts := strings.Fields(s.Text())
		r := R{
			Interface: parts[0],
			Address:   parseHost(parts[1]),
			Gateway:   parseHost(parts[2]),
			Mask:      parseHost(parts[7]),
		}
		r.Metric, _ = strconv.Atoi(parts[6])

		data = append(data, r)
	}
	return data, nil
}

//Netstat gives the list of connections that are known by a system.
func Netstat(ps ...string) ([]C, error) {
	if len(ps) == 0 {
		ps = []string{"tcp", "udp", "tcp", "udp6"}
	}
	vs := make([]C, 0, 24)
	ns := listCommandsBySockets()
	for _, p := range ps {
		switch p {
		case "tcp", "tcp6", "udp", "udp6":
			cs, err := netstat(p, ns)
			if err != nil {
				return nil, err
			}
			vs = append(vs, cs...)
		case "unix":
		default:
			return nil, fmt.Errorf("unknown protocol %s", p)
		}
	}
	return vs, nil
}

func netstat(proto string, ns map[string]string) ([]C, error) {
	f, err := os.Open(filepath.Join(proc, "net", proto))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan()

	data := make([]C, 0, 16)
	for s.Scan() {
		c := C{Proto: proto}
		parts := strings.Fields(s.Text())

		c.Local, c.Remote = parseAddr(parts[1]), parseAddr(parts[2])
		if s, err := strconv.ParseInt(parts[3], 16, 64); err == nil {
			c.State = int(s)
		}

		iob := strings.Split(parts[4], ":")
		c.Recv, _ = strconv.Atoi(iob[0])
		c.Send, _ = strconv.Atoi(iob[1])
		if n, ok := ns[parts[9]]; ok {
			c.Command = n
		} else {
			c.Command = "-"
		}

		data = append(data, c)
	}
	return data, nil
}

func parseHost(h string) string {
	host, _ := hex.DecodeString(h)

	for i := len(host)/2 - 1; i >= 0; i-- {
		j := len(host) - 1 - i
		host[i], host[j] = host[j], host[i]
	}

	return net.IP(host).String()
}

func parseAddr(s string) string {
	h, p, _ := net.SplitHostPort(s)
	port, _ := hex.DecodeString(p)

	if port := int(binary.BigEndian.Uint16(port)); port == 0 {
		p = "*"
	} else {
		p = strconv.Itoa(port)
	}
	return net.JoinHostPort(parseHost(h), p)
}

func listCommandsBySockets() map[string]string {
	const prefix = "socket:"
	is, err := ioutil.ReadDir(proc)
	if err != nil {
		return nil
	}
	vs := make(map[string]string)
	for _, i := range is {
		if !i.IsDir() {
			continue
		}
		p := filepath.Join(proc, i.Name(), "fd")
		is, err := ioutil.ReadDir(p)
		if err != nil {
			continue
		}
		cmd := processName(i.Name(), false)
		for _, i := range is {
			n, err := os.Readlink(filepath.Join(p, i.Name()))
			if err != nil {
				break
			}
			if !strings.HasPrefix(n, prefix) {
				continue
			}
			vs[n[len(prefix)+1:len(n)-1]] = cmd
		}
	}
	return vs
}
