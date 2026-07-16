package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureFileCreatesWithHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")

	require.NoError(t, EnsureFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, Header, string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestEnsureFileLeavesExistingUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o600))

	require.NoError(t, EnsureFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "existing", string(data))
}

func TestEnsureFileParentNotADirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR semantics differ on Windows")
	}
	parent := filepath.Join(t.TempDir(), "afile")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0o600))

	// Statting a path *under* a regular file yields ENOTDIR, which is not
	// ErrNotExist: EnsureFile must surface it, not treat it as "create me".
	err := EnsureFile(filepath.Join(parent, "child.md"))
	require.Error(t, err)
}

func TestAppendWritesTimestampedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")
	const story = "This is a test story."

	require.NoError(t, Append(path, story))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, strings.HasPrefix(content, "\n## "), "entry starts with a timestamp header")
	assert.True(t, strings.HasSuffix(content, "\n\n"+story+"\n"), "entry ends with the story")
}

func TestAppendEmptyStoryIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novel.md")

	require.NoError(t, Append(path, ""))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "no file should be created for an empty story")
}

func TestAppendUnwritablePathReturnsError(t *testing.T) {
	// Parent directory does not exist, so O_CREATE cannot succeed.
	path := filepath.Join(t.TempDir(), "missing-dir", "novel.md")

	err := Append(path, "content")
	require.Error(t, err)
}
