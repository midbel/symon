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

	ARP_LOOPBACK = 772
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
	ARP_LOOPBACK:   "LOCAL",
}

type Socket struct {
	Proto   string `json:"protocol"`
	Local   string `json:"local"`
	Remote  string `json:"remote"`
	State   int    `json:"state"`
	Uid     int    `json:"uid"`
	Recv    int    `json:"recv"`
	Send    int    `json:"send"`
	Command string `json:"command"`
}

func (s Socket) MarshalJSON() ([]byte, error) {
	v := struct {
		Proto   string `json:"protocol"`
		Local   string `json:"local"`
		Remote  string `json:"remote"`
		State   string `json:"state"`
		Recv    int    `json:"recv"`
		Send    int    `json:"send"`
		Command string `json:"command"`
	}{
		Proto:   s.Proto,
		Local:   s.Local,
		Remote:  s.Remote,
		State:   s.Status(),
		Recv:    s.Recv,
		Send:    s.Send,
		Command: s.Command,
	}
	return json.Marshal(v)
}

func (s Socket) User() string {
	u, err := user.LookupId(strconv.Itoa(s.Uid))
	if err == nil {
		return u.Username
	}
	return ""
}

func (s Socket) Status() string {
	if ix := s.State - 1; ix > len(states) {
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
	Label string
	Up    bool
	Mtu   int
	Type  string
	Addr  string
}

//MarshalJSON implementd the json.Marshaler interface.
func (i Interface) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (i Interface) Stats() (Counter, Counter, error) {
	var bs, ps Counter

	qs, err := readProcFile(filepath.Join(proc, "net", "dev"), 17, 2, ' ')
	if err != nil {
		return bs, ps, err
	}
	for rs := range qs {
		if strings.HasPrefix(rs[0], i.Label) {
			ps.In, _ = strconv.ParseFloat(rs[2], 64)
			ps.Out, _ = strconv.ParseFloat(rs[10], 64)

			bs.In, _ = strconv.ParseFloat(rs[1], 64)
			bs.Out, _ = strconv.ParseFloat(rs[9], 64)
			break
		}
	}
	return bs, ps, nil
}

type Addr struct {
	Label string
	net.IP
}

func Addrs() ([]Addr, error) {
	r := filepath.Join(proc, "net", "if_inet6")
	qs, err := readProcFile(r, 6, 0, ' ')
	if err != nil {
		return nil, err
	}
	var as []Addr
	for rs := range qs {
		bs, err := hex.DecodeString(rs[0])
		if err != nil {
			return nil, err
		}
		a := Addr{
			Label: rs[5],
			IP:    net.IP(bs),
		}
		as = append(as, a)
	}
	return as, nil
}

func Interfaces() ([]Interface, error) {
	const p = "/sys/class/net/"
	is, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	ds := make([]Interface, 0, len(is))
	for _, i := range is {
		nic := Interface{Label: i.Name()}
		if bs, err := ioutil.ReadFile(filepath.Join(p, nic.Label, "mtu")); err == nil {
			nic.Mtu, _ = strconv.Atoi(strings.TrimSpace(string(bs)))
		}
		if bs, err := ioutil.ReadFile(filepath.Join(p, nic.Label, "carrier")); err == nil {
			nic.Up = strings.TrimSpace(string(bs)) == "1"
		}
		if bs, err := ioutil.ReadFile(filepath.Join(p, nic.Label, "type")); err == nil {
			t, _ := strconv.Atoi(strings.TrimSpace(string(bs)))
			nic.Type = arpTypes[t]
		}
		if bs, err := ioutil.ReadFile(filepath.Join(p, nic.Label, "address")); err == nil {
			nic.Addr = strings.TrimSpace(string(bs))
		}
		ds = append(ds, nic)
	}
	return ds, nil
}

//Links gives the ARP table used by the kernel for address resolutions.
func Links() ([]Link, error) {
	r := filepath.Join(proc, "net", "arp")
	qs, err := readProcFile(r, 6, 1, ' ')
	if err != nil {
		return nil, err
	}

	ls := make([]Link, 0, 100)
	for rs := range qs {
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
	r := filepath.Join(proc, "net", "route")
	qs, err := readProcFile(r, 11, 1, '\t')
	if err != nil {
		return nil, err
	}
	rs := make([]Route, 0, 16)
	for fs := range qs {
		r := Route{
			Interface: fs[0],
			Address:   parseHost(fs[1]),
			Gateway:   parseHost(fs[2]),
			Mask:      parseHost(fs[7]),
		}
		r.Metric, _ = strconv.Atoi(fs[6])

		rs = append(rs, r)
	}
	return rs, nil
}

//Netstat gives the list of connections that are known by a system.
func Netstat(ps ...string) ([]Socket, error) {
	if len(ps) == 0 {
		ps = []string{"tcp", "udp", "tcp", "udp6"}
	}
	vs := make([]Socket, 0, 24)
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

func netstat(proto string, ns map[string]string) ([]Socket, error) {
	f, err := os.Open(filepath.Join(proc, "net", proto))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan()

	ks := make([]Socket, 0, 16)
	for s.Scan() {
		k := Socket{Proto: proto}
		parts := strings.Fields(s.Text())

		k.Local, k.Remote = parseAddr(parts[1]), parseAddr(parts[2])
		if s, err := strconv.ParseInt(parts[3], 16, 64); err == nil {
			k.State = int(s)
		}

		iob := strings.Split(parts[4], ":")
		k.Recv, _ = strconv.Atoi(iob[0])
		k.Send, _ = strconv.Atoi(iob[1])
		if n, ok := ns[parts[9]]; ok {
			k.Command = n
		} else {
			k.Command = "-"
		}

		ks = append(ks, k)
	}
	return ks, nil
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
