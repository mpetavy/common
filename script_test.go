package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScriptEngine(t *testing.T) {
	InitTesting(t)

	src := "console.log('Hello world!'); 'Done!';"

	otto, err := NewOttoEngine(src)
	if Error(err) {
		return
	}

	goja, err := NewGojaEngine(src)
	if Error(err) {
		return
	}

	vms := []ScriptEngine{otto, goja}

	for _, vm := range vms {
		v, err := vm.Run(time.Millisecond * 250)
		if Error(err) {
			return
		}

		assert.Equal(t, "Done!", v)
	}
}

func TestScriptEngineTimeout(t *testing.T) {
	InitTesting(t)

	src := "while(true) {}"

	otto, err := NewOttoEngine(src)
	if Error(err) {
		return
	}

	goja, err := NewGojaEngine(src)
	if Error(err) {
		return
	}

	vms := []ScriptEngine{otto, goja}

	for _, vm := range vms {
		_, err = vm.Run(time.Millisecond * 500)

		assert.NotNil(t, err, "%T", vm)
		assert.Equal(t, true, IsErrTimeout(err), "%T", vm)
	}
}
