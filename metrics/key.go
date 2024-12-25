package metrics

import (
	"strings"
)

type Key struct {
	Name      string
	Namespace string
	Subsystem string
}

func (k Key) String() string {
	var sb strings.Builder
	sb.WriteString(k.Name)
	if k.Namespace != "" {
		sb.WriteString(".")
		sb.WriteString(k.Namespace)
	}
	if k.Subsystem != "" {
		sb.WriteString(".")
		sb.WriteString(k.Subsystem)
	}
	return sb.String()
}
