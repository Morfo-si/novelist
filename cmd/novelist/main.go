// Command novelist captures timestamped journal entries from the terminal.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/adrg/xdg"

	"github.com/Morfo-si/novelist/internal/prompt"
	"github.com/Morfo-si/novelist/internal/storage"
)

// version is injected at build time via -ldflags "-X main.version=...".
// It falls back to "dev" for local builds without the ldflag set.
var version = "dev"

// novelFile is the journal filename created under the user's Documents dir.
const novelFile = "novelist.md"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses args, drives the prompt, and persists the entry. It returns a
// process exit code and is the seam that keeps main testable.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v", "version":
			fmt.Fprintf(stdout, "novelist %s\n", resolveVersion(version, readBuildInfoVersion()))
			return 0
		}
	}

	path := filepath.Join(xdg.UserDirs.Documents, novelFile)

	var story string
	if err := prompt.New(&story).Run(); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}

	if err := storage.EnsureFile(path); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}
	if err := storage.Append(path, story); err != nil {
		fmt.Fprintf(stderr, "novelist: %v\n", err)
		return 1
	}

	if story != "" {
		fmt.Fprintf(stdout, "Okay. Your thoughts have been saved to %s\n", path)
	}
	return 0
}

// resolveVersion picks the version string to display. An explicit ldflag
// version (anything other than the "dev" default) always wins. Otherwise it
// falls back to the module version recorded in the build info, unless that
// is empty or Go's "(devel)" placeholder for a non-release build.
func resolveVersion(ldflagVersion, buildInfoVersion string) string {
	if ldflagVersion != "dev" {
		return ldflagVersion
	}
	if buildInfoVersion != "" && buildInfoVersion != "(devel)" {
		return buildInfoVersion
	}
	return ldflagVersion
}

// readBuildInfoVersion returns the main module's version as recorded by the
// Go toolchain (e.g. set via `go install pkg@version`), or "" if build info
// is unavailable.
func readBuildInfoVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}
