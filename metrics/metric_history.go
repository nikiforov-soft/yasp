package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metricHistory struct {
	labels        prometheus.Labels
	lastUpdatedAt time.Time
}
