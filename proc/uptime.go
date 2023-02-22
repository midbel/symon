package proc

import (
	"bytes"
	"os"
	"strconv"
	"time"
)

func Uptime() (time.Duration, error) {
	buf, err := os.ReadFile(uptimeFile)
	if err != nil {
		return 0, err
	}
	str, _, ok := bytes.Cut(buf, []byte{0x20})
	if !ok {
		return 0, nil
	}
	sec, err := strconv.ParseFloat(string(str), 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(sec) * time.Second, nil
}

func BootTime() (time.Time, error) {
	var (
		sec, err = Uptime()
		now      = time.Now()
	)
	if err != nil {
		return now, err
	}
	return now.Add(-sec), nil
}
