package metrics

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikiforov-soft/yasp/config"
)

func flattenLabels(keys []string, labels map[string]string) []string {
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, labels[key])
	}
	return result
}

func computeHash(key Key, mapping *config.MetricsMapping, labels prometheus.Labels) string {
	hash := sha256.New()
	hash.Write([]byte(key.String()))
	for i, v := range flattenLabels(mapping.Labels, labels) {
		if i != 0 {
			hash.Write([]byte("_"))
		}
		hash.Write([]byte(v))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
