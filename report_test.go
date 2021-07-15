package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportParser(t *testing.T) {
	t.Run("empty profile", func(t *testing.T) {
		r := strings.NewReader("")
		result, err := parseCoverProfile(r)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("malformed profile", func(t *testing.T) {
		r := strings.NewReader("wrong: thing")
		result, err := parseCoverProfile(r)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("only header", func(t *testing.T) {
		r := strings.NewReader("mode: set")
		result, err := parseCoverProfile(r)

		assert.Equal(t, "set", result.Mode)
		assert.NoError(t, err)
	})

	t.Run("single coverage line", func(t *testing.T) {
		r := strings.NewReader(`mode: set
github.com/flipgroup/module/package/file.go:22.39,24.2 1 1
`)
		result, err := parseCoverProfile(r)

		assert.NotNil(t, result)
		assert.NoError(t, err)
	})

	t.Run("malformed coverage line", func(t *testing.T) {
		r := strings.NewReader(`mode: set
github.com/flipgroup/module/package/file.go:22.39,24.2 1 1
github.com/flipgroup/module/package/file.go:22.39,24.2 1 BLURG
`)
		result, err := parseCoverProfile(r)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("valid coverage lines", func(t *testing.T) {
		r := strings.NewReader(`mode: set
github.com/flipgroup/module/package/file.go:22.39,24.2 5 1
github.com/flipgroup/module/package/file.go:44.39,24.2 3 0
github.com/flipgroup/module/package/file.go:66.39,24.2 2 1
github.com/flipgroup/module/package/file.go:88.39,24.2 1 0
github.com/flipgroup/module/another/file.go:22.39,24.2 20 1
github.com/flipgroup/module/another/file.go:44.39,24.2 40 1
github.com/flipgroup/module/another/file.go:66.39,24.2 60 0
github.com/flipgroup/module/another/file.go:88.39,24.2 80 0
`)
		result, err := parseCoverProfile(r)
		assert.NoError(t, err)

		// verify cover and individual package metrics
		assert.Equal(t, 211, result.Total)
		assert.Equal(t, 67, result.Covered)

		pkg01 := result.Packages["github.com/flipgroup/module/package"]
		assert.Equal(t, 11, pkg01.Total)
		assert.Equal(t, 7, pkg01.Covered)

		pkg02 := result.Packages["github.com/flipgroup/module/another"]
		assert.Equal(t, 200, pkg02.Total)
		assert.Equal(t, 60, pkg02.Covered)
	})
}
