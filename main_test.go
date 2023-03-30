package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModuleName(t *testing.T) {
	assert.Equal(t, "github.com/flipgroup/golang-cover-diff", getModuleName())
}

func TestRelativePackage(t *testing.T) {
	const rootPkgName = "github.com/flipgroup/golang-cover-diff/"

	assert.Equal(t,
		"my/cool/package",
		relativePackage(rootPkgName, "my/cool/package"))

	assert.Equal(t,
		"my/cool/package",
		relativePackage(rootPkgName, "github.com/flipgroup/golang-cover-diff/my/cool/package"))

	assert.Equal(t,
		"my/cool/package/with/a/stupidly/log/package/path/name/keep/going/on/going/plus/s",
		relativePackage(rootPkgName, "github.com/flipgroup/golang-cover-diff/my/cool/package/with/a/stupidly/log/package/path/name/keep/going/on/going/plus/some/more/oh/my/when/will/this/end"))
}

func TestBuildTable(t *testing.T) {
	t.Run("empty data set", func(t *testing.T) {
		base := &CoverProfile{}
		head := &CoverProfile{}

		assert.Equal(t, strings.Trim(`
package                                                                            before    after    delta
-------                                                                           -------  -------  -------
                                                                          total:        -        -      n/a
`, "\n"),
			buildTable("", base, head))
	})

	t.Run("package data only base", func(t *testing.T) {
		base := &CoverProfile{
			Total:   60,
			Covered: 20,
			Packages: map[string]*Package{
				"github.com/flipgroup/golang-cover-diff/my/package": {
					Total:   8,
					Covered: 3,
				},
			},
		}

		head := &CoverProfile{
			Total:   80,
			Covered: 33,
		}

		assert.Equal(t, strings.Trim(`
package                                                                            before    after    delta
-------                                                                           -------  -------  -------
my/package                                                                         37.50%        -     gone
                                                                          total:   33.33%   41.25%   +7.92%
`, "\n"),
			buildTable("github.com/flipgroup/golang-cover-diff", base, head))
	})

	t.Run("package data both sides", func(t *testing.T) {
		base := &CoverProfile{
			Total:   60,
			Covered: 20,
			Packages: map[string]*Package{
				"github.com/flipgroup/golang-cover-diff/my/package": {
					Total:   8,
					Covered: 3,
				},
				"github.com/flipgroup/golang-cover-diff/apples": {
					Total:   52,
					Covered: 17,
				},
			},
		}

		head := &CoverProfile{
			Total:   80,
			Covered: 33,
			Packages: map[string]*Package{
				"github.com/flipgroup/golang-cover-diff/my/package": {
					Total:   28,
					Covered: 16,
				},
				"github.com/flipgroup/golang-cover-diff/apples": {
					Total:   52,
					Covered: 17,
				},
			},
		}

		// note: using `$-$` marker to defeat removal of trailing space from `.editorconfig` settings
		assert.Equal(t, strings.ReplaceAll(strings.Trim(`
package                                                                            before    after    delta
-------                                                                           -------  -------  -------
apples                                                                             32.69%   32.69%      $-$
my/package                                                                         37.50%   57.14%  +19.64%
                                                                          total:   33.33%   41.25%   +7.92%
`, "\n"), "$-$", "   "),
			buildTable("github.com/flipgroup/golang-cover-diff", base, head))
	})
}
