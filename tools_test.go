package main

import (
	"os"
	"testing"
)

// TestWriteFileEmptyContents ensures we can write an empty string ("") to a file,
// validating that writing zero content does not cause an error.
func TestWriteFileEmptyContents(t *testing.T) { 
    temp := "test_empty_write.tmp"
	defer os.Remove(temp)

	// 1. Call write_file with explicit empty contents string ("").
	args := `{"path": "` + temp + `", "contents": ""}`
	
	_, err := write_file(args) // Calls the function under test
	if err != nil {
		t.Fatalf("write_file unexpectedly failed when writing empty content: %v", err)
	}

    // 2. Secondary check: Verify file existence and that the operation succeeded.
    _, err = os.Stat(temp) // Changed to use _ since we only care about 'err' here
    if err != nil {
        t.Fatalf("Failed to stat temporary file: %v", err)
    }
}

func TestWriteFileOverwritesNonEmptyFileWithEmptyContents(t *testing.T) {
	temp := "test_empty_write_overwrite.tmp"
	defer os.Remove(temp)

	if err := os.WriteFile(temp, []byte("existing contents"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := `{"path": "` + temp + `", "contents": ""}`
	if _, err := write_file(args); err != nil {
		t.Fatalf("write_file unexpectedly failed when overwriting with empty content: %v", err)
	}

	info, err := os.Stat(temp)
	if err != nil {
		t.Fatalf("Failed to stat temporary file: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("Expected overwritten file to be empty, got %d bytes", info.Size())
	}
}
