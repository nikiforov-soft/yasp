package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Key struct {
	Name      string
	Namespace string
	Subsystem string
}

func (k Key) String() string {
	return prometheus.BuildFQName(k.Namespace, k.Subsystem, k.Name)
}
