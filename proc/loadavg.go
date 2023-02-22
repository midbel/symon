package proc

import (
	"bytes"
	"os"
	"strconv"
)

func LoadAvg() ([]float64, error) {
	buf, err := os.ReadFile(loadavgFile)
	if err != nil {
		return nil, err
	}
	var list []float64
	for i, b := range bytes.Split(buf, []byte(" ")) {
		if i > 2 {
			break
		}
		f, err := strconv.ParseFloat(string(b), 64)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}
