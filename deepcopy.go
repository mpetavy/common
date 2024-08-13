package common

import (
	"bytes"
	"encoding/gob"
)

func Clone[T any](in T) T {
	buf := new(bytes.Buffer)
	out := new(T)

	Panic(gob.NewEncoder(buf).Encode(in))
	Panic(gob.NewDecoder(buf).Decode(out))

	return *out
}
