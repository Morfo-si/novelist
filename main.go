package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/huh"
)

var UserDirs = xdg.UserDirs.Documents

const (
	NovelFile = "novelist.md"
	UnixDate  = "Mon Jan _2 15:04:05 MST 2006"
)

func main() {
	var story string
	var filePath = fmt.Sprintf("%s/%s", UserDirs, NovelFile)

	huh.NewText().
		Title("Tell me a story.").
		Value(&story).
		Placeholder("What's on your mind?").
		Run()

	t := time.Now()

	// First time creating the file.
	if _, err := os.Stat(filePath); err != nil {
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		} else {
			f.WriteString("# Welcome to Novelist\n")
			f.Close()
		}
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.Write([]byte(fmt.Sprintf("\n## %s\n\n", t.Format(UnixDate))))
	f.Write([]byte(fmt.Sprintf("%s\n", story)))

	fmt.Println("Okay. Your thoughts have been saved to", filePath)
}
