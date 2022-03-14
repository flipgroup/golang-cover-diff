package main

import (
	"strings"
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

func TestBuildTable(t *testing.T) {
	t.Run("empty data set", func(t *testing.T) {
		base := &CoverProfile{}
		head := &CoverProfile{}

		assert.Equal(t, strings.TrimLeft(`
|package                                                                         |  before|   after|   delta|
|-------                                                                         |  ------|   -----|   -----|
|                                                                          total:|       -|       -|     n/a|
`, "\n"),
			buildTable("", base, head))
	})

	t.Run("package data only base", func(t *testing.T) {
		base := &CoverProfile{
			Total:   60,
			Covered: 20,
			Packages: map[string]*Package{
				"github.com/flipgroup/goverdiff/my/package": {
					Total:   8,
					Covered: 3,
				},
			},
		}

		head := &CoverProfile{
			Total:   80,
			Covered: 33,
		}

		assert.Equal(t, strings.TrimLeft(`
|package                                                                         |  before|   after|   delta|
|-------                                                                         |  ------|   -----|   -----|
|my/package                                                                      |  37.50%|       -|    gone|
|                                                                          total:|  33.33%|  41.25%|  +7.92%|
`, "\n"),
			buildTable("github.com/flipgroup/goverdiff", base, head))
	})

	t.Run("package data both sides", func(t *testing.T) {
		base := &CoverProfile{
			Total:   60,
			Covered: 20,
			Packages: map[string]*Package{
				"github.com/flipgroup/goverdiff/my/package": {
					Total:   8,
					Covered: 3,
				},
				"github.com/flipgroup/goverdiff/apples": {
					Total:   52,
					Covered: 17,
				},
			},
		}

		head := &CoverProfile{
			Total:   80,
			Covered: 33,
			Packages: map[string]*Package{
				"github.com/flipgroup/goverdiff/my/package": {
					Total:   28,
					Covered: 16,
				},
				"github.com/flipgroup/goverdiff/apples": {
					Total:   52,
					Covered: 17,
				},
			},
		}

		assert.Equal(t, strings.TrimLeft(`
|package                                                                         |  before|   after|   delta|
|-------                                                                         |  ------|   -----|   -----|
|apples                                                                          |  32.69%|  32.69%|  +0.00%|
|my/package                                                                      |  37.50%|  57.14%| +19.64%|
|                                                                          total:|  33.33%|  41.25%|  +7.92%|
`, "\n"),
			buildTable("github.com/flipgroup/goverdiff", base, head))
	})
}

func TestRenderLinkTemplate(t *testing.T) {
	type kase struct {
		name     string
		tmpl     string
		text     string
		expected string
	}
	ks := []kase{
		{
			name:     "no template specified",
			tmpl:     "",
			text:     "55%",
			expected: "55%",
		},
		{
			name:     "template specified with replace token",
			tmpl:     "https://ci.acme.com/coverprofiles/before/%[p]/coverage.html",
			text:     "55%",
			expected: "[55%](https://ci.acme.com/coverprofiles/before/github.com/flipgroup/goverdiff/coverage.html)",
		},
		{
			name:     "template specified without replace token",
			tmpl:     "https://ci.acme.com/coverprofiles/before/coverage.html",
			text:     "55%",
			expected: "[55%](https://ci.acme.com/coverprofiles/before/coverage.html)",
		},
		{
			name:     "template specified without text",
			tmpl:     "https://ci.acme.com/coverprofiles/before/coverage.html",
			text:     "-",
			expected: "-",
		},
	}

	rep := strings.NewReplacer("%[p]", "github.com/flipgroup/goverdiff")
	for _, k := range ks {
		t.Run(k.name, func(t *testing.T) {
			actual := renderLinkTemplate(k.tmpl, k.text, rep)
			if actual != k.expected {
				t.Errorf("expected: %s, got %s", k.expected, actual)
			}
		})
	}
}
