package template

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var (
	funcsMap = map[string]any{
		"ToLower":    strings.ToLower,
		"ToUpper":    strings.ToUpper,
		"TrimSpaces": strings.TrimSpace,
		"TrimPrefix": strings.TrimPrefix,
		"TrimSuffix": strings.TrimSuffix,
		"ToNumber": func(value any) any {
			var stringValue string
			switch value := value.(type) {
			case bool:
				if value {
					return 1
				}
				return 0
			case uint8:
				return value
			case uint16:
				return value
			case uint32:
				return value
			case uint64:
				return value
			case int8:
				return value
			case int16:
				return value
			case int32:
				return value
			case int64:
				return value
			case float64:
				return value
			case float32:
				return value
			case []byte:
				stringValue = string(value)
			case string:
				stringValue = value
			default:
				return fmt.Sprintf("%v", value)
			}

			if float64Value, err := strconv.ParseFloat(stringValue, 64); err == nil {
				return float64Value
			}

			if int64Value, err := strconv.ParseInt(stringValue, 10, 64); err == nil {
				return int64Value
			}

			return stringValue
		},
		"Split": strings.Split,
		"Last": func(values []string) string {
			if len(values) == 0 {
				return ""
			}
			return values[len(values)-1]
		},
		"Quote": strconv.Quote,
		"JsonMarshal": func(value any) string {
			b, _ := json.Marshal(value)
			return string(b)
		},
		"JsonUnmarshal": func(jsonString string) map[string]any {
			var result map[string]any
			_ = json.Unmarshal([]byte(jsonString), &result)
			return result
		},
	}
)
