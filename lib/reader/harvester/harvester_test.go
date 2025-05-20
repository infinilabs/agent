/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package harvester

import (
	"fmt"
	"infini.sh/agent/lib/reader"
	"io"
	"log"
	"os"
	"testing"
)

// Define the test JSON content as a multiline string or byte slice
const testJSONContent = `
{"timestamp": "2023-01-01T10:00:00Z", "level": "info", "message": "Log entry 1"}
{"timestamp": "2023-01-01T10:00:01Z", "level": "warning", "message": "Log entry 2"}
{"timestamp": "2023-01-01T10:00:02Z", "level": "error", "message": "Log entry 3"}
{"timestamp": "2023-01-01T10:00:03Z", "level": "info", "message": "Log entry 4"}
{"timestamp": "2023-01-01T10:00:04Z", "level": "debug", "message": "Log entry 5"}
{"timestamp": "2023-01-01T10:00:05Z", "level": "info", "message": "Log entry 6"}
{"timestamp": "2023-01-01T10:00:06Z", "level": "warning", "message": "Log entry 7"}
{"timestamp": "2023-01-01T10:00:07Z", "level": "info", "message": "Log entry 8"}
{"timestamp": "2023-01-01T10:00:08Z", "level": "error", "message": "Log entry 9"}
{"timestamp": "2023-01-01T10:00:09Z", "level": "info", "message": "Log entry 10"}
{"timestamp": "2023-01-01T10:00:10Z", "level": "info", "message": "Log entry 11 - final"}
`

// testReadJsonData is a helper function to read from embedded data
func testReadJsonData(t *testing.T) {
	var offset int64 = 0
	// Use a reader for the embedded string content instead of a file path
	// strings.NewReader implements io.ReadSeeker (needed by Harvester or its reader)
	// Assuming NewHarvester or its internal NewJsonFileReader can accept an io.ReadSeeker or similar interface.
	// If NewHarvester MUST take a file path, this method (using embedded data with a Reader) won't work directly.
	// In that case, you would need to use Method 2 (Create Temporary File).

	// *** If NewHarvester needs a path, we must use a temporary file ***
	// Let's switch to Method 2 (Create Temporary File) as it's more likely
	// that NewHarvester is tied to file operations.

	// Create a temporary file
	tempFile, err := CreateTempFileWithContent(testJSONContent)
	if err != nil {
		t.Fatalf("Failed to create temporary file for test: %v", err)
	}
	defer RemoveTempFile(tempFile) // Ensure the temporary file is removed after the test

	// Use the temporary file's path
	filePath := tempFile.Name()

	// Now use the temporary file path with NewHarvester
	h, err := NewHarvester(filePath, offset)
	if err != nil {
		t.Fatalf("Failed to create Harvester: %v", err) // Use t.Fatalf to fail the test
	}
	// Assuming h.Close() is necessary for cleanup, add defer h.Close()
	// The original code didn't close, but it's good practice. Check Harvester's contract.
	// If Harvester manages file handles internally, closing is important.

	// Assuming NewJsonFileReader takes the file path again internally or uses h's state
	r, err := h.NewJsonFileReader("", true) // Check documentation/code for NewJsonFileReader parameters
	if err != nil {
		t.Fatalf("Failed to create JsonFileReader: %v", err) // Use t.Fatalf to fail the test
	}
	// Assuming r needs closing, add defer r.Close() if it has a Close method.

	log.Println("start reading>>>>") // Use t.Log or fmt.Println in tests instead of package log
	t.Log("start reading>>>>")

	var msg reader.Message
	// Read up to 10 messages or until error/EOF
	readCount := 0
	for i := 0; i < 10; i++ { // Loop up to 10 times
		msg, err = r.Next()
		if err != nil {
			// Check for expected EOF
			if err == io.EOF {
				t.Log("Reached end of test data.")
				break
			}
			t.Fatalf("Error reading next message: %v", err) // Use t.Fatalf for unexpected errors
		}
		// t.Log or fmt.Println for output during test
		t.Logf("Message %d: %s", readCount+1, string(msg.Content))
		t.Logf("offset: %d, size: %d, line: %d", msg.Offset, msg.Bytes, msg.LineNumbers)
		readCount++

		// Optional: Add assertions on msg content or offsets if needed
		// For example, check if LineNumbers increases, or if Content is not empty
	}

	// Optional: Add assertions on how many messages were read if expected
	// if readCount != 10 {
	//    t.Errorf("Expected to read 10 messages, but read %d", readCount)
	// }

	// Close resources if they are not automatically managed
	// if r != nil { r.Close() } // If JsonFileReader has a Close method
	// if h != nil { h.Close() } // If Harvester has a Close method
}

// Helper function to create a temporary file with content
func CreateTempFileWithContent(content string) (*os.File, error) {
	// Need to import "os"
	tempFile, err := os.CreateTemp("", "harvester_test_data_*.json") // Create a temp file with a pattern
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	// No need to defer Close here, caller will close it via the returned File object

	_, err = tempFile.WriteString(content)
	if err != nil {
		tempFile.Close()           // Close before returning error
		os.Remove(tempFile.Name()) // Clean up the temp file
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Ensure data is flushed to disk if necessary, though WriteString might do it.
	// if err := tempFile.Sync(); err != nil { ... }

	// Seek back to the beginning of the file if the reader needs to start from the beginning
	_, err = tempFile.Seek(0, io.SeekStart) // Seek to the beginning (offset 0 from the start)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to seek to beginning of temp file: %w", err)
	}

	return tempFile, nil
}

// Helper function to remove a temporary file
func RemoveTempFile(file *os.File) {
	if file == nil {
		return
	}
	file.Close()           // Close the file first
	os.Remove(file.Name()) // Remove the file
}

func TestReader(t *testing.T) {
	// Use t.Run if you want to have subtests, otherwise direct call is fine
	// t.Run("ReadJsonFile", testReadJsonData)
	testReadJsonData(t) // Directly call the test function
}
