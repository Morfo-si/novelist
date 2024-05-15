// Package main is the entry point for the Novelist program.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adrg/xdg"          // Import xdg for cross-platform user directory retrieval.
	"github.com/charmbracelet/huh" // Import huh for CLI text input.
)

// UserDirs specifies the directory where the novel file will be saved.
var UserDirs = xdg.UserDirs.Documents

// Several constants used throughout the program.
const (
	CharLimit = 10 * 1024                      // 10KB is the maximum character limit for the story
	NovelFile = "novelist.md"                  // Name of the novel file.
	UnixDate  = "Mon Jan _2 15:04:05 MST 2006" // Unix date format.
)

// main is the main entry point for the Novelist program.
func main() {
	var story string                                         // Variable to store the user's story.
	var filePath = fmt.Sprintf("%s/%s", UserDirs, NovelFile) // File path for the novel file.

	// Prompt user to tell a story.
	huh.NewText().
		Title("Tell me a story.").
		Value(&story).
		Placeholder("What's on your mind?").
		CharLimit(CharLimit).
		Run()

	t := time.Now() // Current time.

	// Check if the file exists.
	if _, err := os.Stat(filePath); err != nil {
		// If not, create the file and title.
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		} else {
			f.WriteString("# Welcome to Novelist\n")
			f.Close()
		}
	}

	// Open the file for appending.
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Check if any content was provided before saving it
	if len(story) > 0 {
		// Write the current time and the story to the file.
		f.Write([]byte(fmt.Sprintf("\n## %s\n\n", t.Format(UnixDate))))
		f.Write([]byte(fmt.Sprintf("%s\n", story)))

		// Print the file path where the story is saved.
		fmt.Println("Okay. Your thoughts have been saved to", filePath)
	}
}
