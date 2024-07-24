// Package main is the entry point for the Novelist program.
package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	ERRFAILCREATETEMPIR = "Failed to create temporary directory:"
	ERRFAILTOREAD = "Failed to read file:"
	ERRUNEXPECTEDCONTENT = "Unexpected content in file."
)

func TestGeneratePrompt(t *testing.T) {
	var story string
	expected := "Tell me a story"

	prompt := GeneratePrompt(&story)
	assert.Contains(t, prompt.View(), expected, "Prompt doesn't match expected output.")
}

// TestFileExists creates a temporary directory and tests the FileExists function.
func TestFileExists(t *testing.T) {
	// Prepare a temporary directory for testing.
	tmpDir, err := os.MkdirTemp("", "file-exists-test")
	if err != nil {
		t.Fatal(ERRFAILCREATETEMPIR, err)
	}
	defer os.RemoveAll(tmpDir)

	// Set filePath to a file within the temporary directory.
	filePath := tmpDir + "/testfile.txt"

	// Call the FileExists function.
	FileExists(filePath)

	// Check if the file was created.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("File was not created:", err)
	}

	// Check if the file content is correct.
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(ERRFAILTOREAD, err)
	}
	expectedContent := "# Welcome to Novelist\n"
	assert.Equal(t, string(content), expectedContent, ERRUNEXPECTEDCONTENT)
}

// TestFileExistsExistingFile tests the FileExists function when the file already exists.
func TestFileExistsExistingFile(t *testing.T) {
	// Prepare a temporary directory for testing.
	tmpDir, err := os.MkdirTemp("", "file-exists-test")
	if err != nil {
		t.Fatal(ERRFAILCREATETEMPIR, err)
	}
	defer os.RemoveAll(tmpDir)

	// Set filePath to a file within the temporary directory.
	filePath := tmpDir + "/testfile.txt"

	// Create a test file.
	err = os.WriteFile(filePath, []byte("Test content"), 0644)
	if err != nil {
		t.Fatal("Failed to create test file:", err)
	}

	// Call the FileExists function.
	FileExists(filePath)

	// Check if the file content is unchanged.
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(ERRFAILTOREAD, err)
	}
	expectedContent := "Test content"
	assert.Equal(t, string(content), expectedContent, ERRUNEXPECTEDCONTENT)
}

// TestSaveContent tests the SaveContent function.
func TestSaveContent(t *testing.T) {
	// Prepare a temporary directory for testing.
	tmpDir, err := os.MkdirTemp("", "save-content-test")
	if err != nil {
		t.Fatal(ERRFAILCREATETEMPIR, err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the file path for testing.
	testFilePath := tmpDir + "/testfile.md"

	// Prepare test input.
	testStory := "This is a test story."

	// Mock current time.
	testTime := time.Now()

	// Call the SaveContent function.
	SaveContent(testStory, testFilePath)

	// Check if the file was created.
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatal("File was not created:", err)
	}

	// Check if the content was written correctly.
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatal(ERRFAILTOREAD, err)
	}
	expectedContent := "\n## " + testTime.Format(UnixDate) + "\n\n" + testStory + "\n"
	assert.Equal(t, expectedContent, string(content), ERRUNEXPECTEDCONTENT)
}

// TestSaveContentEmptyStory tests the SaveContent function when an empty story is provided.
func TestSaveContentEmptyStory(t *testing.T) {
	// Prepare a temporary directory for testing.
	tmpDir, err := os.MkdirTemp("", "save-content-test")
	if err != nil {
		t.Fatal(ERRFAILCREATETEMPIR, err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the file path for testing.
	testFilePath := tmpDir + "/testfile.md"

	// Call the SaveContent function with an empty story.
	SaveContent("", testFilePath)

	// Check if the file was not created.
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Fatal("File should not have been created.")
	}
}
