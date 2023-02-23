package common

import (
	"encoding/json"
)

func Clone[T any](t T) (T, error) {
	x := new(T)

	ba, err := json.Marshal(t)
	if Error(err) {
		return *x, err
	}

	err = json.Unmarshal(ba, x)
	if Error(err) {
		return *x, err
	}

	return *x, nil
}
