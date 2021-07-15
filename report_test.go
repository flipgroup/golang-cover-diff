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

		assert.NotNil(t, result)
		assert.NoError(t, err)
	})
}
