package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzAppend asserts the round-trip invariant: after appending a non-empty
// story the file ends with that story, and arbitrary input never panics.
// Lower-value than a security fuzz target (Novelist has no injection surface),
// but the round-trip property is real.
func FuzzAppend(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("multi\nline\n## looks like a header")

	f.Fuzz(func(t *testing.T, story string) {
		path := filepath.Join(t.TempDir(), "novel.md")
		if err := EnsureFile(path); err != nil {
			t.Fatalf("EnsureFile: %v", err)
		}
		if err := Append(path, story); err != nil {
			t.Fatalf("Append: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if story != "" && !strings.HasSuffix(string(data), story+"\n") {
			t.Fatalf("file does not end with the appended story")
		}
	})
}
