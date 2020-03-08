package common

import (
	"encoding/json"
)

func DeepCopy(dst interface{}, src interface{}) error {
	DebugFunc()

	ba, err := json.Marshal(src)
	if Error(err) {
		return err
	}

	err = json.Unmarshal(ba, dst)
	if Error(err) {
		return err
	}

	return nil
}
