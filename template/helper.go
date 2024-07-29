package template

import (
	"fmt"
	"strconv"
	"strings"
)

func AsNumber(data []byte) (float64, error) {
	if strings.ToLower(string(data)) == "true" {
		return 1, nil
	}
	if strings.ToLower(string(data)) == "false" {
		return 0, nil
	}
	if float64Value, err := strconv.ParseFloat(string(data), 64); err == nil {
		return float64Value, nil
	}
	if int64Value, err := strconv.ParseInt(string(data), 10, 64); err == nil {
		return float64(int64Value), nil
	}
	return 0, fmt.Errorf("failed to parse %s as number", string(data))
}

func IsNumber(n []byte) bool {
	if len(n) > 0 && n[0] == '-' {
		n = n[1:]
	}
	if len(n) == 0 {
		return false
	}

	var hasPoint bool
	for _, c := range n {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && len(n) > 1 && !hasPoint {
			hasPoint = true
			continue
		}
		return false
	}
	return true
}
