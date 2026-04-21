// Package main is the entry point for the Novelist program.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"charm.land/huh/v2"   // Import huh for CLI text input.
	"github.com/adrg/xdg" // Import xdg for cross-platform user directory retrieval.
)

// version is injected at build time via -ldflags "-X main.version=..."
// It falls back to "dev" for local builds without the ldflag set.
var version = "dev"

// UserDirs specifies the directory where the novel file will be saved.
var (
	FilePath = fmt.Sprintf("%s/%s", UserDirs, NovelFile) // File path for the novel file.
	UserDirs = xdg.UserDirs.Documents
)

// Several constants used throughout the program.
const (
	CharLimit = 10 * 1024                      // 10KB is the maximum character limit for the story
	NovelFile = "novelist.md"                  // Name of the novel file.
	UnixDate  = "Mon Jan _2 15:04:05 MST 2006" // Unix date format.
)

// GeneratePrompt returns a NewText instance.
func GeneratePrompt(story *string) *huh.Text {
	return huh.NewText().
		Title("Tell me a story.").
		Value(story).
		Placeholder("What's on your mind?").
		CharLimit(CharLimit)
}

// main is the main entry point for the Novelist program.
func main() {
	// Handle --version / -v as a pre-flight check so users can verify the
	// version that was injected at build time.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("novelist %s\n", version)
			return
		}
	}

	var story string // Variable to store the user's story.

	// Prompt user to tell a story.
	prompt := GeneratePrompt(&story)
	prompt.Run()

	// Check if the file exists.
	FileExists(FilePath)

	// Save the content.
	SaveContent(story, FilePath)
}

// FileExists checks if a file exists to save the content.
func FileExists(f string) {
	// If not, create the file and add a title.
	if _, err := os.Stat(f); err != nil {

		f, err := os.Create(f)
		if err != nil {
			log.Fatal(err)
		}
		f.WriteString("# Welcome to Novelist\n")
		f.Close()
	}
}

func SaveContent(story string, f string) {
	t := time.Now() // Current time.

	// Check if any content was provided before saving it
	if len(story) > 0 {
		// Open the file for appending.
		file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// Write the current time and the story to the file.
		file.Write(fmt.Appendf(nil, "\n## %s\n\n", t.Format(UnixDate)))
		file.Write(fmt.Appendf(nil, "%s\n", story))

		// Print the file path where the story is saved.
		fmt.Printf("Okay. Your thoughts have been saved to %v", f)
	}
}
