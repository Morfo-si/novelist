// Package storage persists Novelist journal entries to a Markdown file.
package storage

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	// Header is written to a freshly created journal file.
	Header = "# Welcome to Novelist\n"
	// dateFormat is the Unix date layout used to stamp each entry.
	dateFormat = "Mon Jan _2 15:04:05 MST 2006"
	// filePerm keeps the journal private: it is a personal diary, so it must
	// not be world-readable.
	filePerm = 0o600
)

// EnsureFile creates path with the welcome Header if it does not already
// exist. It is a no-op when the file is present. A stat error that is not
// "file does not exist" (for example a permission problem) is returned rather
// than silently treated as absence.
func EnsureFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	//nolint:gosec // G304: path is Novelist's own journal file under the user's
	// Documents directory, not attacker-controlled input.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(Header); err != nil {
		return fmt.Errorf("write header to %s: %w", path, err)
	}
	return nil
}

// Append writes a timestamped entry containing story to path. It is a no-op
// when story is empty, preserving the original "save nothing for empty input"
// behavior.
func Append(path, story string) error {
	if story == "" {
		return nil
	}

	//nolint:gosec // G304: path is Novelist's own journal file, not
	// attacker-controlled input.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	entry := fmt.Sprintf("\n## %s\n\n%s\n", time.Now().Format(dateFormat), story)
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("append to %s: %w", path, err)
	}
	return nil
}
