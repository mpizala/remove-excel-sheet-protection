package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mpizala/remove-excel-sheet-protection/internal/xlsx"
)

func main() {
	// Check arguments
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: RESP <file.xlsx>")
		os.Exit(2)
	}

	filePath := os.Args[1]

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(filePath), ".xlsx") {
		fmt.Fprintf(os.Stderr, "Error: file must have .xlsx extension\n")
		os.Exit(1)
	}

	// Convert to absolute path for better error messages
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", absPath)
		os.Exit(1)
	}

	// Check if file is readable and writable
	file, err := os.OpenFile(absPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Error: permission denied: %s\n", absPath)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot access file: %s\n", err)
		}
		os.Exit(1)
	}
	file.Close()

	// Remove protection
	if err := xlsx.RemoveProtection(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Silent success (Unix philosophy)
	os.Exit(0)
}
