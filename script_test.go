package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScriptEngine(t *testing.T) {
	InitTesting(t)

	//src := "console.log('Hello world!'); 'Done!';"
	src := "let person = {firstName:'John', lastName:'Doe', age:50, eyeColor:'blue'};console.table(person);'Done!';"

	engine, err := NewScriptEngine(src)
	if Error(err) {
		return
	}

	vms := []*ScriptEngine{engine}

	for _, vm := range vms {
		v, err := vm.Run(time.Millisecond*250, "", "")
		if Error(err) {
			return
		}

		assert.Equal(t, "Done!", v)
	}
}

func TestScriptEngineTimeout(t *testing.T) {
	InitTesting(t)

	src := "while(true) {}"

	engine, err := NewScriptEngine(src)
	if Error(err) {
		return
	}

	vms := []*ScriptEngine{engine}

	for _, vm := range vms {
		_, err = vm.Run(time.Millisecond*500, "", "")

		assert.NotNil(t, err, "%T", vm)
		assert.Equal(t, true, IsErrTimeout(err), "%T", vm)
	}
}
