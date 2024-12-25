package metrics

func flattenLabels(keys []string, labels map[string]string) []string {
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, labels[key])
	}
	return result
}
