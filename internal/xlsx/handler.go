package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Regex pattern to match sheetProtection elements
// Matches both self-closing <sheetProtection .../> and explicit closing tags
var sheetProtectionRegex = regexp.MustCompile(`<sheetProtection[^>]*(?:/>|>.*?</sheetProtection>)`)

// RemoveProtection removes sheet protection from an Excel (.xlsx) file
// It modifies the file in-place by:
// 1. Extracting the ZIP archive
// 2. Removing <sheetProtection> elements from worksheet XML files
// 3. Repackaging the ZIP with the same structure
// 4. Replacing the original file
//
// Returns nil on success (even if no protection was found)
func RemoveProtection(filePath string) error {
	// Read the entire ZIP file into memory
	zipData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Open as ZIP archive
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("invalid or corrupted .xlsx file: %w", err)
	}

	// Create a buffer for the new ZIP file
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Track if any modifications were made
	modified := false

	// Process each file in the ZIP archive
	for _, file := range zipReader.File {
		if err := processZipFile(file, zipWriter, &modified); err != nil {
			zipWriter.Close()
			return fmt.Errorf("failed to process %s: %w", file.Name, err)
		}
	}

	// Close the ZIP writer
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize ZIP: %w", err)
	}

	// Write to a temporary file first for atomic replacement
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename (replace original file)
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	return nil
}

// processZipFile processes a single file from the ZIP archive
// If it's a worksheet XML file, it removes sheet protection
// Otherwise, it copies the file unchanged
func processZipFile(file *zip.File, zipWriter *zip.Writer, modified *bool) error {
	// Read the file content
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	// Check if this is a worksheet XML file
	isWorksheet := strings.HasPrefix(file.Name, "xl/worksheets/") &&
		strings.HasSuffix(file.Name, ".xml") &&
		strings.Contains(file.Name, "sheet")

	// Process worksheet files to remove protection
	if isWorksheet {
		originalLen := len(content)
		content = sheetProtectionRegex.ReplaceAll(content, []byte{})
		if len(content) != originalLen {
			*modified = true
		}
	}

	// Create file header preserving compression method and metadata
	header := &zip.FileHeader{
		Name:   file.Name,
		Method: file.Method, // Preserve compression (Store or Deflate)
	}
	header.SetModTime(file.Modified)

	// Write the file to the new ZIP
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create header: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// IsProtected checks if an Excel file has any sheet protection
// This is useful for testing or informational purposes
func IsProtected(filePath string) (bool, error) {
	zipData, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return false, fmt.Errorf("invalid or corrupted .xlsx file: %w", err)
	}

	// Check each worksheet file
	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/") &&
			strings.HasSuffix(file.Name, ".xml") {

			reader, err := file.Open()
			if err != nil {
				continue
			}

			content, err := io.ReadAll(reader)
			reader.Close()

			if err != nil {
				continue
			}

			if sheetProtectionRegex.Match(content) {
				return true, nil
			}
		}
	}

	return false, nil
}
