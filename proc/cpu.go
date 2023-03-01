package proc

import (
	"bufio"
	"os"
	"strings"

	"github.com/midbel/slices"
)

type CpuInfo struct {
	Ident     string
	User      int
	Nice      int
	Sys       int
	Idle      int
	Iowait    int
	Irq       int
	SoftIrq   int
	Steal     int
	Guest     int
	GuestNice int
}

func Cpu() ([]CpuInfo, error) {
	return readSystemStat(statFile)
}

func readSystemStat(file string) ([]CpuInfo, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var (
		scan = bufio.NewScanner(r)
		list []CpuInfo
	)
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if !strings.HasPrefix(slices.Fst(fields), "cpu") {
			continue
		}
		cpu := CpuInfo{
			Ident: slices.Fst(fields),
		}
		list = append(list, cpu)
	}
	return list, nil
}
