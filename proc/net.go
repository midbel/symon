package proc

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/midbel/slices"
)

type ConnState byte

const (
	StateNull ConnState = iota
	StateEstablished
	StateSynSent
	StateSynRecv
	StateFinWait1
	StateFinWait2
	StateTimeWait
	StateClose
	StateCloseWait
	StateLastAck
	StateListen
	StateClosing
)

func (c ConnState) String() string {
	switch c {
	default:
		return ""
	case StateEstablished:
		return "ESTABLISHED"
	case StateSynSent:
		return "SYN SENT"
	case StateSynRecv:
		return "SYN RECV"
	case StateFinWait1:
		return "FIN WAIT1"
	case StateFinWait2:
		return "FIN WAIT2"
	case StateTimeWait:
		return "TIME WAIT"
	case StateClose:
		return "CLOSE"
	case StateCloseWait:
		return "CLOSE WAIT"
	case StateLastAck:
		return "LAST ACK"
	case StateListen:
		return "LISTEN"
	case StateClosing:
		return "CLOSING"
	}
}

type ConnInfo struct {
	Proto  string
	Local  netip.AddrPort
	Remote netip.AddrPort
	State  ConnState
	User   string
}

func Netstat() ([]ConnInfo, error) {
	list, err := Tcp()
	if err != nil {
		return nil, err
	}
	rest, err := Udp()
	if err != nil {
		return nil, err
	}
	return append(list, rest...), nil
}

func Tcp() ([]ConnInfo, error) {
	conns, err := readSocketTable(tcpFile)
	if err != nil {
		return nil, err
	}
	rest, err := readSocketTable(tcp6File)
	if err != nil {
		return nil, err
	}
	return append(conns, rest...), nil
}

func Udp() ([]ConnInfo, error) {
	conns, err := readSocketTable(udpFile)
	if err != nil {
		return nil, err
	}
	rest, err := readSocketTable(udp6File)
	if err != nil {
		return nil, err
	}
	return append(conns, rest...), nil
}

type RouteInfo struct {
	Interface string
	Mask      netip.Prefix
	Network   netip.Addr
	Gateway   netip.Addr
}

func Routes() ([]RouteInfo, error) {
	r, err := os.Open(routeFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	getAddr := func(str string) (netip.Addr, error) {
		tmp, err := hex.DecodeString(str)
		if err != nil {
			return netip.Addr{}, err
		}
		ip, ok := netip.AddrFromSlice(slices.Reverse(tmp))
		if !ok {
			return ip, fmt.Errorf("invalid address")
		}
		return ip, nil
	}

	var (
		list []RouteInfo
		scan = bufio.NewScanner(r)
	)
	scan.Scan()
	for scan.Scan() {
		var (
			line   = strings.TrimSpace(scan.Text())
			fields = strings.Fields(line)
			route  RouteInfo
			err    error
		)
		route.Interface = slices.At(fields, 0)
		if route.Network, err = getAddr(slices.At(fields, 1)); err != nil {
			return nil, err
		}
		if route.Gateway, err = getAddr(slices.At(fields, 1)); err != nil {
			return nil, err
		}
		list = append(list, route)
	}
	return list, scan.Err()
}

func readSocketTable(file string) ([]ConnInfo, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	getAddrPort := func(str string) (netip.AddrPort, error) {
		var (
			addr, num, _ = strings.Cut(str, ":")
			port, err    = strconv.ParseUint(num, 16, 16)
		)
		if err != nil {
			return netip.AddrPort{}, err
		}
		tmp, err := hex.DecodeString(addr)
		if err != nil {
			return netip.AddrPort{}, err
		}
		tmp = slices.Reverse(tmp)
		ip, ok := netip.AddrFromSlice(tmp)
		if !ok {
			return netip.AddrPort{}, fmt.Errorf("invalid address")
		}
		return netip.AddrPortFrom(ip, uint16(port)), nil
	}

	var (
		scan  = bufio.NewScanner(r)
		proto = filepath.Base(file)
		list  []ConnInfo
	)
	scan.Scan()
	for scan.Scan() {
		var (
			line   = strings.TrimSpace(scan.Text())
			fields = strings.Fields(line)
			conn   = ConnInfo{
				Proto: proto,
			}
			err error
		)
		if conn.Local, err = getAddrPort(slices.At(fields, 1)); err != nil {
			return nil, err
		}
		if conn.Remote, err = getAddrPort(slices.At(fields, 2)); err != nil {
			return nil, err
		}

		state, err := strconv.ParseInt(slices.At(fields, 3), 16, 8)
		if err != nil {
			return nil, err
		}
		conn.State = ConnState(state)

		who, err := user.LookupId(slices.At(fields, 7))
		if err != nil {
			return nil, err
		}
		conn.User = who.Username

		list = append(list, conn)
	}
	return list, scan.Err()
}
