package xiaomi

import (
	"fmt"
	"strings"
)

func formatMacAddress(macAddress string) string {
	if strings.Contains(macAddress, ":") {
		return macAddress
	}
	return strings.ToUpper(fmt.Sprintf("%s:%s:%s:%s:%s:%s", macAddress[0:2], macAddress[2:4], macAddress[4:6], macAddress[6:8], macAddress[8:10], macAddress[10:12]))
}
