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

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		name             string
		ldflagVersion    string
		buildInfoVersion string
		want             string
	}{
		{"explicit ldflag wins", "v1.2.3", "v9.9.9", "v1.2.3"},
		{"falls back to build info", "dev", "v0.0.8", "v0.0.8"},
		{"no build info keeps dev", "dev", "", "dev"},
		{"devel placeholder keeps dev", "dev", "(devel)", "dev"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveVersion(tc.ldflagVersion, tc.buildInfoVersion)
			assert.Equal(t, tc.want, got)
		})
	}
}
