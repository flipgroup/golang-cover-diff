package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModulePackageName(t *testing.T) {
	assert.Equal(t, "github.com/flipgroup/goverdiff", getModulePackageName())
}

func TestRelativePackage(t *testing.T) {
	const rootPkgName = "github.com/flipgroup/goverdiff/"

	assert.Equal(t,
		"my/cool/package",
		relativePackage(rootPkgName, "my/cool/package"))

	assert.Equal(t,
		"my/cool/package",
		relativePackage(rootPkgName, "github.com/flipgroup/goverdiff/my/cool/package"))

	assert.Equal(t,
		"my/cool/package/with/a/stupidly/log/package/path/name/keep/going/on/going/plus/s",
		relativePackage(rootPkgName, "github.com/flipgroup/goverdiff/my/cool/package/with/a/stupidly/log/package/path/name/keep/going/on/going/plus/some/more/oh/my/when/will/this/end"))
}
