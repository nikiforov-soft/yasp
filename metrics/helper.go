package metrics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

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

func computeHash(mapping *config.MetricsMapping, labels prometheus.Labels) string {
	hash := sha256.New()
	hash.Write([]byte(mapping.Namespace))
	hash.Write([]byte(","))
	hash.Write([]byte(mapping.Subsystem))
	hash.Write([]byte(","))
	hash.Write([]byte(mapping.Name))
	if len(mapping.Labels) != 0 {
		hash.Write([]byte(","))
		for i, v := range flattenLabels(mapping.Labels, labels) {
			if i != 0 {
				hash.Write([]byte("_"))
			}
			hash.Write([]byte(v))
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validateLabels(key Key, mapping *config.MetricsMapping, labels prometheus.Labels) error {
	if len(labels) != len(mapping.Labels) {
		return fmt.Errorf("mismatched label name/value count for %s expected %d got %d", key, len(mapping.Labels), len(labels))
	}

	for _, k := range mapping.Labels {
		if _, exists := labels[k]; !exists {
			return fmt.Errorf("missing required label %s for: %s", k, key)
		}
	}

	for k := range labels {
		if !slices.Contains(mapping.Labels, k) {
			return fmt.Errorf("provided unknown label %s for: %s", k, key)
		}
	}
	return nil
}
