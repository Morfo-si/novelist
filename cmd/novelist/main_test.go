package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunVersionFlag(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		var out bytes.Buffer
		code := run([]string{arg}, &out, io.Discard)
		assert.Equal(t, 0, code, "arg %q exits 0", arg)
		assert.Contains(t, out.String(), "novelist ", "arg %q prints the version", arg)
	}
}
