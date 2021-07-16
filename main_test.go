package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModulePackageName(t *testing.T) {
	assert.Equal(t, "github.com/flipgroup/goverdiff", getModulePackageName())
}
