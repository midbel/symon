package symon

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

const blocksPath = "/sys/class/blocks/"

type Filesystem struct {
	Label   string   `json:"label"`
	Point   string   `json:"point"`
	Type    string   `json:"type"`
	Options []string `json:"options"`
	Dump    int      `json:"dump"`
	Check   int      `json:"check"`
	Size    int      `json:"size"`
}

//Mount gives the list of filesystem currently mounted on a system.
func Mount() ([]Filesystem, error) {
	r := filepath.Join(proc, "mounts")
	qs, err := readProcFile(r, 6, 0, ' ')
	if err != nil {
		return nil, err
	}
	var fs []Filesystem
	for rs := range qs {
		f := Filesystem{
			Label:   rs[0],
			Point:   rs[1],
			Type:    rs[2],
			Options: strings.Split(rs[3], ","),
		}
		f.Dump, _ = strconv.Atoi(rs[4])
		f.Check, _ = strconv.Atoi(rs[5])

		if bs, err := ioutil.ReadFile(filepath.Join(blocksPath, f.Label, "size")); err == nil {
			s := strings.TrimSpace(string(bs))
			f.Size, _ = strconv.Atoi(s)
		}

		fs = append(fs, f)
	}
	return fs, nil
}
