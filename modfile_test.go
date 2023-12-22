package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var content []byte = []byte(` 
module github.com/author/app

go 1.21

toolchain go1.21.1

require (
github.com/author0/lib0 v0
github.com/author1/lib1 v1
)

require (
github.com/author2/lib2 v2 // indirect
github.com/author3/lib3 v3 // indirect
)

replace github.com/author0/lib0 => ../lib0

exclude github.com/author4/lib4 v4

retract [v1.0.0,v1.0.5] // Build broken on some platforms.
`)

func TestModfile(t *testing.T) {
	InitTesting(t)

	mf, err := ReadModfile(content)
	Error(err)

	assert.Equal(t, "app", mf.Title())
	assert.Equal(t, "github.com/author/app", mf.Module.Name)

	assert.Equal(t, "1.21", mf.GO.Version)
	assert.Equal(t, "go1.21.1", mf.Toolchain.Version)

	requires := []Require{
		{"github.com/author0/lib0", "v0", false},
		{"github.com/author1/lib1", "v1", false},
		{"github.com/author2/lib2", "v2", true},
		{"github.com/author3/lib3", "v3", true},
	}

	for i := 0; i < len(requires); i++ {
		assert.Equal(t, requires[i].Module, mf.Requires[i].Module)
		assert.Equal(t, requires[i].Version, mf.Requires[i].Version)
		assert.Equal(t, requires[i].Indirect, mf.Requires[i].Indirect)
	}

	replaces := []Replace{
		{"github.com/author0/lib0", "../lib0"},
	}

	for i := 0; i < len(replaces); i++ {
		assert.Equal(t, replaces[i].Module, mf.Replaces[i].Module)
		assert.Equal(t, replaces[i].Target, mf.Replaces[i].Target)
	}

	excludes := []Exclude{
		{"github.com/author4/lib4", "v4"},
	}

	for i := 0; i < len(excludes); i++ {
		assert.Equal(t, excludes[i].Module, mf.Excludes[i].Module)
		assert.Equal(t, excludes[i].Version, mf.Excludes[i].Version)
	}

	retracts := []Retract{
		{"v1.0.0", "v1.0.5", "Build broken on some platforms."},
	}

	for i := 0; i < len(retracts); i++ {
		assert.Equal(t, retracts[i].VersionLow, mf.Retracts[i].VersionLow)
		assert.Equal(t, retracts[i].VersionHigh, mf.Retracts[i].VersionHigh)
		assert.Equal(t, retracts[i].Comment, mf.Retracts[i].Comment)
	}
}
