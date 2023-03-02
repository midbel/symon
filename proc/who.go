package proc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

type Who struct {
	Type     int
	Pid      int
	Terminal string
	TermId   string
	User     string
	Host     string

	TermStatus int
	ExitStatus int

	Session int
	When    time.Time
	Addr    netip.Addr
}

func (w Who) Regular() bool {
	if w.User == "" {
		return false
	}
	_, err := user.Lookup(w.User)
	return err == nil
}

func (w Who) Command() string {
	var (
		pid      = strconv.Itoa(w.Pid)
		comm     = filepath.Join(proc, pid, "comm")
		buf, err = os.ReadFile(comm)
	)
	if err != nil {
		return ""
	}
	buf = bytes.Trim(buf, "\x00")
	buf = bytes.TrimSpace(buf)
	return string(buf)
}

func Current() ([]Who, error) {
	return readWho(utmpFile)
}

func All() ([]Who, error) {
	return readWho(wtmpFile)
}

const (
	recordSize = 384
	lineSize   = 32
	nameSize   = 32
	hostSize   = 256
	addrSize   = 16
)

func readWho(file string) ([]Who, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var (
		buf  = make([]byte, recordSize)
		list []Who
	)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if n != recordSize {
			return nil, fmt.Errorf("not enough bytes read for one record (%d != %d)", n, recordSize)
		}
		who, err := parseWho(bytes.NewReader(buf))
		if err != nil {
			return nil, err
		}
		list = append(list, who)
	}
	return list, nil
}

func parseWho(r io.Reader) (Who, error) {

	readString := func(size int) string {
		tmp := make([]byte, size)
		io.ReadFull(r, tmp)
		tmp = bytes.Trim(tmp, "\x00")
		return string(tmp)
	}

	readAddr := func() netip.Addr {
		var tmp [addrSize]byte
		io.ReadFull(r, tmp[:])
		return netip.AddrFrom16(tmp)
	}

	readTime := func() time.Time {
		var (
			val  int32
			when time.Time
		)
		binary.Read(r, binary.LittleEndian, &val)
		when = time.Unix(int64(val), 0)
		binary.Read(r, binary.LittleEndian, &val)
		return when.Add(time.Microsecond * time.Duration(val))
	}

	var (
		who   Who
		short int16
		long  int32
	)

	binary.Read(r, binary.LittleEndian, &long)
	who.Type = int(long)
	binary.Read(r, binary.LittleEndian, &long)
	who.Pid = int(long)

	who.Terminal = readString(lineSize)
	who.TermId = readString(4)
	who.User = readString(nameSize)
	who.Host = readString(hostSize)

	binary.Read(r, binary.LittleEndian, &short)
	who.TermStatus = int(short)
	binary.Read(r, binary.LittleEndian, &short)
	who.ExitStatus = int(short)
	binary.Read(r, binary.LittleEndian, &long)
	who.Session = int(long)

	who.When = readTime()
	who.Addr = readAddr()

	return who, nil
}
