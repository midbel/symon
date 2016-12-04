package symon

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
	Proto  string `json:"protocol"`
	Local  string `json:"local_address"`
	Remote string `json:"remote_address"`
	State  int    `json:"state"`
	Uid    int    `json:"uid"`
	Recv   int    `json:"recv"`
	Send   int    `json:"send"`
}

func (c C) Status() string {
	if c.Proto != "tcp" {
		return ""
	}
	if ix := c.State - 1; ix > len(states) {
		return "unknown"
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
func Route() ([]R, error) {
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
func Netstat(proto ...string) ([]C, error) {
	if len(proto) == 0 {
		proto = []string{"tcp", "udp"}
	}
	data := make([]C, 0, 24)
	for _, p := range proto {
		switch p {
		case "tcp", "tcp6", "udp", "udp6":
			cs, err := netstat(p)
			if err != nil {
				return nil, err
			}
			data = append(data, cs...)
		case "unix":
		default:
			return nil, fmt.Errorf("unknown protocol %s", p)
		}
	}
	return data, nil
}

func netstat(proto string) ([]C, error) {
	f, err := os.Open(filepath.Join(proc, "net", proto))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan()

	data := make([]C, 0, 16)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}
		c := C{Proto: proto}
		parts := strings.Fields(s.Text())

		c.Local, c.Remote = parseAddr(parts[1]), parseAddr(parts[2])
		if s, err := strconv.ParseInt(parts[3], 16, 64); err == nil {
			c.State = int(s)
		}
		iob := strings.Split(parts[4], ":")
		c.Recv, _ = strconv.Atoi(iob[0])
		c.Send, _ = strconv.Atoi(iob[1])

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
