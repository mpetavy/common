package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMin(t *testing.T) {
	assert.Equal(t, -5, Min(-5, 0, 5))
}

func TestMax(t *testing.T) {
	assert.Equal(t, 5, Max(-5, 0, 5))
}
