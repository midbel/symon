package symon

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type F struct {
	Label   string   `json:"label"`
	Point   string   `json:"point"`
	Type    string   `json:"type"`
	Options []string `json:"options"`
	Dump    int      `json:"dump"`
	Check   int      `json:"check"`
}

//Mount gives the list of filesystem currently mounted on a system.
func Mount() ([]F, error) {
	f, err := os.Open(filepath.Join(proc, "mounts"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data := make([]F, 0, 16)

	s := bufio.NewScanner(f)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}
		parts := strings.Fields(s.Text())
		f := F{}
		for i, field := range []interface{}{&f.Label, &f.Point, &f.Type, &f.Options, &f.Dump, &f.Check} {
			switch field := field.(type) {
			case *int:
				*field, _ = strconv.Atoi(parts[i])
			case *string:
				*field = parts[i]
			case *[]string:
				values := strings.Split(parts[i], ",")
				options := make([]string, len(values))
				for i, v := range values {
					options[i] = strings.TrimSpace(v)
				}
				*field = options
			}
		}
		data = append(data, f)
	}
	return data, nil
}
