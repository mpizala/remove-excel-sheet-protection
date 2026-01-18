# RESP - Remove Excel Sheet Protection

A lightweight CLI tool to remove sheet protection from Excel (.xlsx) files.

## Features

- Removes sheet protection from all worksheets in an Excel file
- Modifies files in-place (overwrites original)
- Fast and lightweight (uses only Go standard library)
- Cross-platform (macOS, Linux, Windows)
- Silent operation following Unix philosophy
- Preserves all Excel data and formatting

## Installation

### Homebrew (macOS/Linux)

```bash
brew install mpizala/utils/resp
```

### From Source

Requires Go 1.21 or later:

```bash
go install github.com/mpizala/remove-excel-sheet-protection@latest
```

Or build manually:

```bash
git clone https://github.com/mpizala/remove-excel-sheet-protection.git
cd remove-excel-sheet-protection
go build -o resp
```

## Usage

```bash
RESP file.xlsx
```

The tool operates silently on success. The original file is modified in-place.

### Examples

Remove protection from a single file:
```bash
RESP protected-spreadsheet.xlsx
```

Process multiple files:
```bash
for file in *.xlsx; do RESP "$file"; done
```

## How It Works

Excel (.xlsx) files are ZIP archives containing XML files. Sheet protection is stored in `xl/worksheets/*.xml` files as `<sheetProtection>` XML elements. RESP:

1. Opens the Excel file as a ZIP archive
2. Identifies worksheet XML files
3. Removes `<sheetProtection>` elements using regex
4. Repackages the ZIP archive
5. Replaces the original file atomically

## Exit Codes

- `0` - Success (protection removed or no protection found)
- `1` - Error (file not found, corrupted file, permission denied)
- `2` - Invalid arguments (no file specified or wrong extension)

## Limitations

- Only works with `.xlsx` files (Excel 2007+)
- Does not work with `.xls` files (older Excel format)
- Removes protection but does not recover passwords
- Requires read/write permissions on the file

## Security Note

Excel sheet protection is not encryption. It's a simple UI restriction that prevents casual editing. This tool removes that restriction. Use responsibly and only on files you have permission to modify.

## Development

### Run Tests

```bash
go test ./... -v
```

### Build

```bash
go build -o resp
```

### Run Locally

```bash
./resp test.xlsx
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! Please open an issue or pull request.

## Repository

https://github.com/mpizala/remove-excel-sheet-protection
