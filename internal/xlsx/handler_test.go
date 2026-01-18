package xlsx

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// createTestXLSX creates a minimal valid XLSX file for testing
func createTestXLSX(worksheetContent string) []byte {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// [Content_Types].xml - required for valid XLSX
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`
	writeZipFile(zipWriter, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	writeZipFile(zipWriter, "_rels/.rels", rels)

	// xl/workbook.xml
	workbook := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sheet1" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`
	writeZipFile(zipWriter, "xl/workbook.xml", workbook)

	// xl/_rels/workbook.xml.rels
	workbookRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`
	writeZipFile(zipWriter, "xl/_rels/workbook.xml.rels", workbookRels)

	// xl/worksheets/sheet1.xml (with custom content)
	writeZipFile(zipWriter, "xl/worksheets/sheet1.xml", worksheetContent)

	zipWriter.Close()
	return buf.Bytes()
}

// writeZipFile writes a file to a ZIP archive
func writeZipFile(zipWriter *zip.Writer, name, content string) {
	writer, _ := zipWriter.Create(name)
	writer.Write([]byte(content))
}

// createTempXLSX creates a temporary XLSX file for testing
func createTempXLSX(t *testing.T, content []byte) string {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.xlsx")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return filePath
}

func TestRemoveProtection_WithProtection(t *testing.T) {
	// Create worksheet with sheet protection
	worksheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetProtection password="83AF" sheet="1" objects="1" scenarios="1"/>
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>Test Data</v></c>
    </row>
  </sheetData>
</worksheet>`

	xlsxData := createTestXLSX(worksheetContent)
	filePath := createTempXLSX(t, xlsxData)

	// Verify protection exists before removal
	protected, err := IsProtected(filePath)
	if err != nil {
		t.Fatalf("IsProtected failed: %v", err)
	}
	if !protected {
		t.Fatal("Expected file to be protected before removal")
	}

	// Remove protection
	if err := RemoveProtection(filePath); err != nil {
		t.Fatalf("RemoveProtection failed: %v", err)
	}

	// Verify protection is removed
	protected, err = IsProtected(filePath)
	if err != nil {
		t.Fatalf("IsProtected failed after removal: %v", err)
	}
	if protected {
		t.Fatal("Expected protection to be removed")
	}

	// Verify file is still valid ZIP
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("File is not a valid ZIP after processing: %v", err)
	}
}

func TestRemoveProtection_WithoutProtection(t *testing.T) {
	// Create worksheet without sheet protection
	worksheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>Test Data</v></c>
    </row>
  </sheetData>
</worksheet>`

	xlsxData := createTestXLSX(worksheetContent)
	filePath := createTempXLSX(t, xlsxData)

	// Remove protection (should succeed even without protection)
	if err := RemoveProtection(filePath); err != nil {
		t.Fatalf("RemoveProtection failed: %v", err)
	}

	// Verify file is still valid
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("File is not a valid ZIP after processing: %v", err)
	}
}

func TestRemoveProtection_ComplexProtection(t *testing.T) {
	// Test various protection formats
	testCases := []struct {
		name       string
		protection string
	}{
		{
			name:       "Self-closing with attributes",
			protection: `<sheetProtection password="83AF" sheet="1" objects="1" scenarios="1" formatCells="0" formatColumns="0"/>`,
		},
		{
			name:       "Self-closing minimal",
			protection: `<sheetProtection sheet="1"/>`,
		},
		{
			name:       "With explicit closing tag",
			protection: `<sheetProtection sheet="1"></sheetProtection>`,
		},
		{
			name:       "With namespace prefix",
			protection: `<x:sheetProtection xmlns:x="http://schemas.openxmlformats.org/spreadsheetml/2006/main" sheet="1"/>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			worksheetContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  ` + tc.protection + `
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>Test</v></c>
    </row>
  </sheetData>
</worksheet>`

			xlsxData := createTestXLSX(worksheetContent)
			filePath := createTempXLSX(t, xlsxData)

			// Remove protection
			if err := RemoveProtection(filePath); err != nil {
				t.Fatalf("RemoveProtection failed: %v", err)
			}

			// Verify protection is removed
			protected, err := IsProtected(filePath)
			if err != nil {
				t.Fatalf("IsProtected failed: %v", err)
			}
			if protected {
				t.Fatal("Expected protection to be removed")
			}
		})
	}
}

func TestRemoveProtection_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.xlsx")

	// Create a non-ZIP file
	if err := os.WriteFile(filePath, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should return an error
	err := RemoveProtection(filePath)
	if err == nil {
		t.Fatal("Expected error for invalid ZIP file")
	}
}

func TestRemoveProtection_NonExistentFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "nonexistent.xlsx")

	// Should return an error
	err := RemoveProtection(filePath)
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

func TestIsProtected(t *testing.T) {
	testCases := []struct {
		name               string
		worksheetContent   string
		expectedProtection bool
	}{
		{
			name: "Protected with password",
			worksheetContent: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetProtection password="83AF" sheet="1"/>
  <sheetData/>
</worksheet>`,
			expectedProtection: true,
		},
		{
			name: "Protected without password",
			worksheetContent: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetProtection sheet="1"/>
  <sheetData/>
</worksheet>`,
			expectedProtection: true,
		},
		{
			name: "Not protected",
			worksheetContent: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData/>
</worksheet>`,
			expectedProtection: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			xlsxData := createTestXLSX(tc.worksheetContent)
			filePath := createTempXLSX(t, xlsxData)

			protected, err := IsProtected(filePath)
			if err != nil {
				t.Fatalf("IsProtected failed: %v", err)
			}

			if protected != tc.expectedProtection {
				t.Errorf("Expected protection=%v, got %v", tc.expectedProtection, protected)
			}
		})
	}
}

func TestRegexPattern(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Self-closing tag",
			input:    `<worksheet><sheetProtection sheet="1"/><sheetData/></worksheet>`,
			expected: `<worksheet><sheetData/></worksheet>`,
		},
		{
			name:     "Explicit closing tag",
			input:    `<worksheet><sheetProtection sheet="1"></sheetProtection><sheetData/></worksheet>`,
			expected: `<worksheet><sheetData/></worksheet>`,
		},
		{
			name:     "With attributes",
			input:    `<worksheet><sheetProtection password="83AF" sheet="1" objects="1"/><sheetData/></worksheet>`,
			expected: `<worksheet><sheetData/></worksheet>`,
		},
		{
			name:     "No protection",
			input:    `<worksheet><sheetData/></worksheet>`,
			expected: `<worksheet><sheetData/></worksheet>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sheetProtectionRegex.ReplaceAllString(tc.input, "")
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}
