package flatten

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Flattener handles the flattening of TypeScript files by inlining local imports
type Flattener struct {
	// processedFiles tracks files that have already been processed to prevent infinite loops
	processedFiles map[string]bool
	// baseDir is the directory of the entry file, used for resolving relative paths
	baseDir string
	// externalImports tracks all external imports to deduplicate them
	externalImports map[string]bool
}

// NewFlattener creates a new Flattener instance
func NewFlattener() *Flattener {
	return &Flattener{
		processedFiles:  make(map[string]bool),
		externalImports: make(map[string]bool),
	}
}

// FlattenFile reads a TypeScript file and recursively inlines all local imports
// Returns the flattened content as a string
func (f *Flattener) FlattenFile(entryFile string) (string, error) {
	// Convert to absolute path and set base directory
	absPath, err := filepath.Abs(entryFile)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", entryFile, err)
	}

	f.baseDir = filepath.Dir(absPath)

	// Start the recursive flattening process
	inlinedContent, err := f.flattenFileRecursive(absPath, "")
	if err != nil {
		return "", err
	}

	// Build the final result with external imports at the top
	var result strings.Builder

	// Write all external imports first (deduplicated)
	for importStatement := range f.externalImports {
		result.WriteString(importStatement)
		result.WriteString("\n")
	}

	// Add a blank line between imports and content if there are external imports
	if len(f.externalImports) > 0 {
		result.WriteString("\n")
	}

	// Write the inlined content
	result.WriteString(inlinedContent)

	return result.String(), nil
}

// flattenFileRecursive recursively processes a single file and inlines its local imports
func (f *Flattener) flattenFileRecursive(filePath string, relativePath string) (string, error) {
	// Normalize the file path
	normalizedPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to normalize path %s: %w", filePath, err)
	}

	// Check if this file has already been processed (cycle detection)
	if f.processedFiles[normalizedPath] {
		return fmt.Sprintf("// Skipping %s (already processed - avoiding cycle)\n", relativePath), nil
	}

	// Mark this file as being processed
	f.processedFiles[normalizedPath] = true

	// Read the file content
	content, err := os.ReadFile(normalizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse imports from the file
	imports, contentWithoutImports := f.parseImports(string(content))

	// Build the result
	var result strings.Builder

	// Add a comment with the file path (relative to base directory if possible)
	var displayPath string
	if relativePath != "" {
		displayPath = relativePath
	} else {
		if relPath, err := filepath.Rel(f.baseDir, normalizedPath); err == nil {
			displayPath = relPath
		} else {
			displayPath = normalizedPath
		}
	}
	result.WriteString(fmt.Sprintf("// File: %s\n", displayPath))

	// Process each import and inline local imports
	for _, importInfo := range imports {
		if f.isLocalImport(importInfo.Path) {
			// Resolve the import path relative to the current file
			importPath := f.resolveImportPath(filepath.Dir(normalizedPath), importInfo.Path)

			// Get relative path for display
			var relativeImportPath string
			if relPath, err := filepath.Rel(f.baseDir, importPath); err == nil {
				relativeImportPath = relPath
			} else {
				relativeImportPath = importPath
			}

			// Recursively process the imported file
			inlinedContent, err := f.flattenFileRecursive(importPath, relativeImportPath)
			if err != nil {
				return "", fmt.Errorf("failed to process import %s: %w", importInfo.Path, err)
			}

			result.WriteString(inlinedContent)
			result.WriteString("\n")
		} else {
			// Collect external imports for deduplication
			f.externalImports[importInfo.Statement] = true
		}
	}

	// Add the file content without import statements
	result.WriteString(contentWithoutImports)

	return result.String(), nil
}

// ImportInfo represents information about an import statement
type ImportInfo struct {
	Statement string // The full import statement
	Path      string // The import path (what's inside quotes)
}

// parseImports extracts import statements from TypeScript content
// Returns a slice of ImportInfo and the content with import statements removed
func (f *Flattener) parseImports(content string) ([]ImportInfo, string) {
	var imports []ImportInfo

	// Regex patterns for different types of import statements
	patterns := []*regexp.Regexp{
		// import { something } from 'path'
		regexp.MustCompile(`import\s+\{[^}]*\}\s+from\s+['"]([^'"]+)['"];?`),
		// import something from 'path'
		regexp.MustCompile(`import\s+\w+\s+from\s+['"]([^'"]+)['"];?`),
		// import * as something from 'path'
		regexp.MustCompile(`import\s+\*\s+as\s+\w+\s+from\s+['"]([^'"]+)['"];?`),
		// import 'path'
		regexp.MustCompile(`import\s+['"]([^'"]+)['"];?`),
		// import type { something } from 'path'
		regexp.MustCompile(`import\s+type\s+\{[^}]*\}\s+from\s+['"]([^'"]+)['"];?`),
		// import type something from 'path'
		regexp.MustCompile(`import\s+type\s+\w+\s+from\s+['"]([^'"]+)['"];?`),
	}

	contentWithoutImports := content

	// Process each pattern
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				imports = append(imports, ImportInfo{
					Statement: strings.TrimSpace(match[0]),
					Path:      match[1],
				})

				// Remove this import from the content
				contentWithoutImports = strings.ReplaceAll(contentWithoutImports, match[0], "")
			}
		}
	}

	// Clean up extra newlines
	contentWithoutImports = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(contentWithoutImports, "\n\n")
	contentWithoutImports = strings.TrimLeft(contentWithoutImports, "\n")

	return imports, contentWithoutImports
}

// isLocalImport determines if an import path is local (relative)
func (f *Flattener) isLocalImport(importPath string) bool {
	// Local imports start with . or ..
	return strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") || importPath == "." || importPath == ".."
}

// resolveImportPath resolves a relative import path to an absolute file path
func (f *Flattener) resolveImportPath(currentDir, importPath string) string {
	// Join the current directory with the import path
	resolved := filepath.Join(currentDir, importPath)

	// Try different extensions if the path doesn't exist
	extensions := []string{"", ".ts", ".tsx", ".js", ".jsx"}

	for _, ext := range extensions {
		candidate := resolved + ext
		if f.fileExists(candidate) {
			return candidate
		}
	}

	// If it's a directory, try index files
	indexFiles := []string{"index.ts", "index.tsx", "index.js", "index.jsx"}
	for _, indexFile := range indexFiles {
		candidate := filepath.Join(resolved, indexFile)
		if f.fileExists(candidate) {
			return candidate
		}
	}

	// Return the original resolved path if nothing is found
	// This will likely cause an error later, but that's expected behavior
	return resolved + ".ts" // Default to .ts extension
}

// fileExists checks if a file exists
func (f *Flattener) fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
