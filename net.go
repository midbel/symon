package symon

import (
	"bufio"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
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

const (
	ARP_NETROM     = 0
	ARP_ETHER      = 1
	ARP_EETHER     = 2
	ARP_AX25       = 3
	ARP_PRONET     = 4
	ARP_CHAOS      = 5
	ARP_IEEE802    = 6
	ARP_ARCNET     = 7
	ARP_APPLETLK   = 8
	ARP_DLCI       = 15
	ARP_ATM        = 19
	ARP_METRICOM   = 23
	ARP_IEEE1394   = 24
	ARP_EUI64      = 27
	ARP_INFINIBAND = 32
)

var arpTypes = map[int]string{
	ARP_NETROM:     "NETROM",
	ARP_ETHER:      "ETHER",
	ARP_EETHER:     "EETHER",
	ARP_AX25:       "AX25",
	ARP_PRONET:     "PRONET",
	ARP_CHAOS:      "CHAOS",
	ARP_IEEE802:    "IEEE802",
	ARP_ARCNET:     "ARCNET",
	ARP_APPLETLK:   "APPLETLK",
	ARP_DLCI:       "DLCI",
	ARP_ATM:        "ATM",
	ARP_METRICOM:   "METRICOM",
	ARP_IEEE1394:   "IEEE1394",
	ARP_EUI64:      "EUI64",
	ARP_INFINIBAND: "INFINIBAND",
}

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

type Route struct {
	Interface string `json:"interface"`
	Address   string `json:"address"`
	Gateway   string `json:"gateway"`
	Mask      string `json:"mask"`
	Metric    int    `json:"metric"`
	Distance  int    `json:"distance"`
}

type Link struct {
	Interface string `json:"interface"`
	Address   string `json:"address"`
	Hardware  string `json:"hardware"`
	Type      string `json:"type"`
	Mask      string `json:"mask"`
}

type Interface struct {
	Label string `json:"interface"`
	SendP int64  `json:"tx-packets"`
	SendB int64  `json:"tx-bytes"`
	RecvP int64  `json:"rx-packets"`
	RecvB int64  `json:"rx-bytes"`
}

func Interfaces() ([]Interface, error) {
	f, err := os.Open(filepath.Join(proc, "net", "dev"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	r.ReadString('\n')
	r.ReadString('\n')

	c := csv.NewReader(r)
	c.Comma = ' '
	c.FieldsPerRecord = 17
	c.TrimLeadingSpace = true

	ds := make([]Interface, 0, 16)
	for rs, err := c.Read(); err == nil; rs, err = c.Read() {
		if ix := strings.Index(rs[0], ":"); ix >= 0 {
			rs[0] = rs[0][:ix]
		}
		i := Interface{Label: rs[0]}
		i.SendB, _ = strconv.ParseInt(rs[1], 10, 64)
		i.SendP, _ = strconv.ParseInt(rs[2], 10, 64)
		i.RecvB, _ = strconv.ParseInt(rs[9], 10, 64)
		i.RecvP, _ = strconv.ParseInt(rs[10], 10, 64)

		ds = append(ds, i)
	}
	return ds, nil
}

func Links() ([]Link, error) {
	f, err := os.Open(filepath.Join(proc, "net", "arp"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	if _, err := r.ReadString('\n'); err != nil {
		return nil, err
	}

	c := csv.NewReader(r)
	c.Comma = ' '
	c.FieldsPerRecord = 6
	c.TrimLeadingSpace = true

	if _, err := c.Read(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	ls := make([]Link, 0, 100)
	for rs, err := c.Read(); err == nil; rs, err = c.Read() {
		t, _ := strconv.ParseInt(rs[1], 0, 8)
		i := Link{
			Interface: rs[5],
			Address:   rs[0],
			Hardware:  rs[3],
			Mask:      rs[4],
			Type:      arpTypes[int(t)],
		}
		ls = append(ls, i)
	}
	return ls, nil
}

//Route gives the list of network routes currently known by a system.
func Routes() ([]Route, error) {
	f, err := os.Open(filepath.Join(proc, "net", "route"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan()

	rs := make([]Route, 0, 16)
	for s.Scan() {
		fs := strings.Fields(s.Text())
		r := Route{
			Interface: fs[0],
			Address:   parseHost(fs[1]),
			Gateway:   parseHost(fs[2]),
			Mask:      parseHost(fs[7]),
		}
		r.Metric, _ = strconv.Atoi(fs[6])

		rs = append(rs, r)
	}
	return rs, s.Err()
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
