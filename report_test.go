package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoverProfileCoverage(t *testing.T) {
	var cp *CoverProfile
	assert.Equal(t, -1, cp.Coverage())

	cp = &CoverProfile{}
	cp.Total = 100
	cp.Covered = 50
	assert.Equal(t, 5000, cp.Coverage())

	cp.Total = 0
	cp.Covered = 50
	assert.Equal(t, -1, cp.Coverage())

	cp.Total = 100
	cp.Covered = 0
	assert.Equal(t, 0, cp.Coverage())
}

func TestPackageCoverage(t *testing.T) {
	var pkg *Package
	assert.Equal(t, -1, pkg.Coverage())

	pkg = &Package{}
	pkg.Total = 100
	pkg.Covered = 50
	assert.Equal(t, 5000, pkg.Coverage())

	pkg.Total = 0
	pkg.Covered = 50
	assert.Equal(t, -1, pkg.Coverage())

	pkg.Total = 100
	pkg.Covered = 0
	assert.Equal(t, 0, pkg.Coverage())
}

func TestReportParser(t *testing.T) {
	t.Run("empty profile", func(t *testing.T) {
		r := strings.NewReader("")
		profile, err := parseCoverProfile(r)

		assert.Nil(t, profile)
		assert.Error(t, err)
	})

	t.Run("malformed profile", func(t *testing.T) {
		r := strings.NewReader("wrong: thing")
		profile, err := parseCoverProfile(r)

		assert.Nil(t, profile)
		assert.Error(t, err)
	})

	t.Run("only header", func(t *testing.T) {
		r := strings.NewReader("mode: set")
		profile, err := parseCoverProfile(r)

		assert.Equal(t, "set", profile.Mode)
		assert.NoError(t, err)
	})

	t.Run("single coverage line", func(t *testing.T) {
		r := strings.NewReader(`mode: set
github.com/flipgroup/module/package/file.go:22.39,24.2 1 1
`)
		profile, err := parseCoverProfile(r)

		assert.NotNil(t, profile)
		assert.NoError(t, err)
	})

	t.Run("malformed coverage line", func(t *testing.T) {
		r := strings.NewReader(`mode: set
github.com/flipgroup/module/package/file.go:22.39,24.2 1 1
github.com/flipgroup/module/package/file.go:22.39,24.2 1 BLURG
`)
		profile, err := parseCoverProfile(r)

		assert.Nil(t, profile)
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
		profile, err := parseCoverProfile(r)
		assert.NoError(t, err)

		// verify cover and individual package metrics
		assert.Equal(t, 211, profile.Total)
		assert.Equal(t, 67, profile.Covered)
		assert.Equal(t, 3175, profile.Coverage()) // equal to 31.75%

		pkg01 := profile.Packages["github.com/flipgroup/module/package"]
		assert.Equal(t, 11, pkg01.Total)
		assert.Equal(t, 7, pkg01.Covered)
		assert.Equal(t, 6363, pkg01.Coverage()) // equal to 63.63%

		pkg02 := profile.Packages["github.com/flipgroup/module/another"]
		assert.Equal(t, 200, pkg02.Total)
		assert.Equal(t, 60, pkg02.Covered)
		assert.Equal(t, 3000, pkg02.Coverage()) // equal to 30%
	})
}
