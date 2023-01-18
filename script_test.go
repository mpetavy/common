package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScriptEngine(t *testing.T) {
	InitTesting(t)

	vm, err := NewScriptEngine("console.log('Hello world!'); 'Done!';")
	if Error(err) {
		return
	}

	v, err := vm.Run(time.Second)
	if Error(err) {
		return
	}

	assert.Equal(t, "Done!", v.String())
}

func TestScriptEngineTimeout(t *testing.T) {
	InitTesting(t)

	vm, err := NewScriptEngine("while(true) {}")
	if Error(err) {
		return
	}

	_, err = vm.Run(time.Second)

	assert.NotNil(t, err)
	assert.Equal(t, true, IsErrTimeout(err))
}
