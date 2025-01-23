package common

import (
	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/require"
	"slices"
	"strings"
	"testing"
)

func TestJSONCompare(t *testing.T) {
	json1 := []byte(`{
		"name": "Alice",
		"age": 30,
		"location": "NY",
		"meta": { "version": 1, "flag": true, "timestamp": 1700000000 },
		"items": [ { "id": 1, "price": 100 }, { "id": 2, "price": 200 } ]
	}`)

	json2 := []byte(`{
		"name": "Alice",
		"age": 31,
		"location": "NY",
		"meta": { "version": 2, "flag": false, "timestamp": 1800000000 },
		"items": [ { "id": 1, "price": 150 }, { "id": 2, "price": 250 } ]
	}`)

	skipPaths := []string{
		"age",
		"meta.version",
		"items.price",
	}

	shouldSkip := func(keyPath string, value any, depth int) bool {
		// Ignore keys by name
		if slices.Contains(skipPaths, keyPath) {
			return true
		}

		// Ignore fields with "timestamp" in their name
		if strings.Contains(keyPath, "timestamp") {
			return true
		}

		// Ignore keys if they have an integer value greater than 1000
		if num, ok := value.(float64); ok && num > 1000 {
			return true
		}

		// Ignore deeply nested keys (example: depth > 2)
		if depth > 3 {
			return true
		}

		return false
	}

	diff, _, err := JSONCompare(json1, json2, jsondiff.DefaultJSONOptions(), shouldSkip)
	require.NoError(t, err)
	require.Equal(t, diff.String(), jsondiff.NoMatch.String())

	skipPaths = append(skipPaths, "meta.flag")

	diff, _, err = JSONCompare(json1, json2, jsondiff.DefaultJSONOptions(), shouldSkip)
	require.NoError(t, err)
	require.Equal(t, diff.String(), jsondiff.FullMatch.String())
}
