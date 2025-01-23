package common

import (
	"encoding/json"
	"github.com/nsf/jsondiff"
)

type IgnoreCondition func(keyPath string, value any, depth int) bool

func removeKeysRecursively(data any, prefix string, depth int, shouldSkip IgnoreCondition) any {
	switch v := data.(type) {
	case map[string]any:
		filtered := make(map[string]any)
		for key, value := range v {
			fullKey := key

			if prefix != "" {
				fullKey = prefix + "." + key
			}

			if shouldSkip(fullKey, value, depth) {
				continue
			}

			filtered[key] = removeKeysRecursively(value, fullKey, depth+1, shouldSkip)
		}
		return filtered

	case []any:
		filteredArray := make([]any, len(v))

		for i, elem := range v {
			filteredArray[i] = removeKeysRecursively(elem, prefix, depth+1, shouldSkip)
		}

		return filteredArray

	default:
		return v
	}
}

// convertJSONToMap unmarshals a JSON string into a generic map[string]any structure.
func convertJSONToMap(jsonStr []byte) (map[string]any, error) {
	var data map[string]any

	err := json.Unmarshal(jsonStr, &data)
	if Error(err) {
		return nil, err
	}

	return data, nil
}

func JSONCompare(json1, json2 []byte, options jsondiff.Options, ignoreCondition IgnoreCondition) (jsondiff.Difference, string, error) {
	DebugFunc()

	obj1, err := convertJSONToMap(json1)
	if Error(err) {
		return jsondiff.NoMatch, "", err
	}

	obj2, err := convertJSONToMap(json2)
	if Error(err) {
		return jsondiff.NoMatch, "", err
	}

	obj1Filtered := removeKeysRecursively(obj1, "", 0, ignoreCondition)
	obj2Filtered := removeKeysRecursively(obj2, "", 0, ignoreCondition)

	jsonBytes1, err := json.Marshal(obj1Filtered)
	if Error(err) {
		return jsondiff.NoMatch, "", err
	}

	jsonBytes2, err := json.Marshal(obj2Filtered)
	if Error(err) {
		return jsondiff.NoMatch, "", err
	}

	diff, txt := jsondiff.Compare(jsonBytes1, jsonBytes2, &options)
	if Error(err) {
		return jsondiff.NoMatch, "", err
	}

	return diff, txt, nil
}
