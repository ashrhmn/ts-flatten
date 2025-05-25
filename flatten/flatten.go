package flatten

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportDetails represents detailed information about an import
type ImportDetails struct {
	ModulePath     string            // The module being imported from
	ImportType     string            // "named", "default", "namespace", "side-effect", "type-named", "type-default"
	DefaultImport  string            // For default imports
	NamedImports   map[string]string // For named imports (imported name -> alias, or imported name -> imported name if no alias)
	NamespaceAlias string            // For namespace imports (import * as name)
}

// Flattener handles the flattening of TypeScript files by inlining local imports
type Flattener struct {
	// processedFiles tracks files that have already been processed to prevent infinite loops
	processedFiles map[string]bool
	// baseDir is the directory of the entry file, used for resolving relative paths
	baseDir string
	// rootDir is the root directory for resolving path-mapped imports (defaults to baseDir)
	rootDir string
	// externalImports tracks external imports grouped by module path for merging
	externalImports map[string]*ImportDetails
	// additionalLocalPrefixes contains additional prefixes to treat as local imports
	additionalLocalPrefixes []string
}

// NewFlattener creates a new Flattener instance
func NewFlattener() *Flattener {
	return &Flattener{
		processedFiles:  make(map[string]bool),
		externalImports: make(map[string]*ImportDetails),
	}
}

// NewFlattenerWithPrefixes creates a new Flattener instance with additional local prefixes
func NewFlattenerWithPrefixes(additionalPrefixes []string) *Flattener {
	return &Flattener{
		processedFiles:          make(map[string]bool),
		externalImports:         make(map[string]*ImportDetails),
		additionalLocalPrefixes: additionalPrefixes,
	}
}

// SetRootDir sets the root directory for resolving path-mapped imports
func (f *Flattener) SetRootDir(rootDir string) {
	f.rootDir = rootDir
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

	// Set rootDir to baseDir if not already set
	if f.rootDir == "" {
		f.rootDir = f.baseDir
	}

	// Start the recursive flattening process
	inlinedContent, err := f.flattenFileRecursive(absPath, "")
	if err != nil {
		return "", err
	}

	// Build the final result with external imports at the top
	var result strings.Builder

	// Write all external imports first (merged and deduplicated)
	for _, importDetails := range f.externalImports {
		importStatement := f.generateImportStatement(importDetails)
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
			// Collect external imports for merging and deduplication
			f.addExternalImport(importInfo)
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

// isLocalImport determines if an import path is local (relative or matching additional prefixes)
func (f *Flattener) isLocalImport(importPath string) bool {
	// Local imports start with . or ..
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") || importPath == "." || importPath == ".." {
		return true
	}

	// Check against additional local prefixes
	for _, prefix := range f.additionalLocalPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}

	return false
}

// resolveImportPath resolves an import path to an absolute file path
func (f *Flattener) resolveImportPath(currentDir, importPath string) string {
	var resolved string

	// Check if this is a relative import (starts with . or ..)
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") || importPath == "." || importPath == ".." {
		// Join the current directory with the import path
		resolved = filepath.Join(currentDir, importPath)
	} else {
		// This is likely an additional local prefix import
		// Try to resolve it relative to the root directory
		resolved = filepath.Join(f.rootDir, importPath)
	}

	// Try different extensions if the path doesn't exist
	extensions := []string{"", ".ts", ".tsx", ".d.ts", ".js", ".jsx"}

	for _, ext := range extensions {
		candidate := resolved + ext
		if f.fileExists(candidate) {
			return candidate
		}
	}

	// If it's a directory, try index files
	indexFiles := []string{"index.ts", "index.tsx", "index.d.ts", "index.js", "index.jsx"}
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

// addExternalImport adds an external import, merging with existing imports from the same module
func (f *Flattener) addExternalImport(importInfo ImportInfo) {
	details := f.parseImportStatement(importInfo.Statement, importInfo.Path)

	// Normalize module path (handle node: prefix)
	normalizedPath := f.normalizeModulePath(details.ModulePath)
	details.ModulePath = normalizedPath

	if existing, exists := f.externalImports[normalizedPath]; exists {
		// Merge with existing import from same module
		f.mergeImportDetails(existing, details)
	} else {
		// Check for naming conflicts with other modules before adding
		f.resolveNamingConflicts(details)
		// New module, add it
		f.externalImports[normalizedPath] = details
	}
}

// parseImportStatement parses an import statement into ImportDetails
func (f *Flattener) parseImportStatement(statement, path string) *ImportDetails {
	details := &ImportDetails{
		ModulePath:   path,
		NamedImports: make(map[string]string),
	}

	// Determine import type and parse accordingly
	if strings.Contains(statement, "import type") {
		if strings.Contains(statement, "{") {
			details.ImportType = "type-named"
			f.parseNamedImports(statement, details)
		} else {
			details.ImportType = "type-default"
			f.parseDefaultImport(statement, details)
		}
	} else if strings.Contains(statement, "* as") {
		details.ImportType = "namespace"
		f.parseNamespaceImport(statement, details)
	} else if strings.Contains(statement, "{") {
		details.ImportType = "named"
		f.parseNamedImports(statement, details)
	} else if !strings.Contains(statement, "from") {
		details.ImportType = "side-effect"
	} else {
		details.ImportType = "default"
		f.parseDefaultImport(statement, details)
	}

	return details
}

// parseNamedImports extracts named imports from the import statement
func (f *Flattener) parseNamedImports(statement string, details *ImportDetails) {
	// Extract the part between { and }
	re := regexp.MustCompile(`\{([^}]*)\}`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) > 1 {
		imports := strings.Split(matches[1], ",")
		for _, imp := range imports {
			imp = strings.TrimSpace(imp)
			if imp != "" {
				// Handle aliases (e.g., "name as alias")
				if strings.Contains(imp, " as ") {
					parts := strings.Split(imp, " as ")
					if len(parts) == 2 {
						details.NamedImports[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				} else {
					details.NamedImports[imp] = imp
				}
			}
		}
	}
}

// parseDefaultImport extracts default import from the import statement
func (f *Flattener) parseDefaultImport(statement string, details *ImportDetails) {
	// Extract default import name (between import and from)
	re := regexp.MustCompile(`import\s+(?:type\s+)?(\w+)\s+from`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) > 1 {
		details.DefaultImport = matches[1]
	}
}

// parseNamespaceImport extracts namespace import from the import statement
func (f *Flattener) parseNamespaceImport(statement string, details *ImportDetails) {
	// Extract namespace alias (after "* as")
	re := regexp.MustCompile(`\*\s+as\s+(\w+)`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) > 1 {
		details.NamespaceAlias = matches[1]
	}
}

// mergeImportDetails merges two ImportDetails from the same module
func (f *Flattener) mergeImportDetails(existing, new *ImportDetails) {
	// Merge default imports (prefer existing, warn if different)
	if new.DefaultImport != "" && existing.DefaultImport == "" {
		existing.DefaultImport = new.DefaultImport
	}

	// Merge namespace imports (prefer existing, warn if different)
	if new.NamespaceAlias != "" && existing.NamespaceAlias == "" {
		existing.NamespaceAlias = new.NamespaceAlias
	}

	// Merge named imports (deduplicate)
	for name, alias := range new.NamedImports {
		existing.NamedImports[name] = alias
	}

	// If new import has a different type, prioritize more specific types
	typeOrder := map[string]int{
		"side-effect":  0,
		"default":      1,
		"named":        2,
		"namespace":    3,
		"type-default": 4,
		"type-named":   5,
	}

	if typeOrder[new.ImportType] > typeOrder[existing.ImportType] {
		existing.ImportType = new.ImportType
	}
}

// generateImportStatement generates an import statement from ImportDetails
func (f *Flattener) generateImportStatement(details *ImportDetails) string {
	var parts []string

	switch details.ImportType {
	case "side-effect":
		return fmt.Sprintf(`import "%s";`, details.ModulePath)

	case "namespace":
		return fmt.Sprintf(`import * as %s from "%s";`, details.NamespaceAlias, details.ModulePath)

	case "type-default":
		return fmt.Sprintf(`import type %s from "%s";`, details.DefaultImport, details.ModulePath)

	case "type-named":
		namedParts := f.formatNamedImports(details.NamedImports)
		return fmt.Sprintf(`import type { %s } from "%s";`, namedParts, details.ModulePath)

	default: // "default", "named", or mixed
		if details.DefaultImport != "" {
			parts = append(parts, details.DefaultImport)
		}

		if len(details.NamedImports) > 0 {
			namedParts := f.formatNamedImports(details.NamedImports)
			parts = append(parts, fmt.Sprintf("{ %s }", namedParts))
		}

		if len(parts) > 0 {
			return fmt.Sprintf(`import %s from "%s";`, strings.Join(parts, ", "), details.ModulePath)
		}
	}

	return ""
}

// formatNamedImports formats named imports with aliases
func (f *Flattener) formatNamedImports(namedImports map[string]string) string {
	var imports []string
	for name, alias := range namedImports {
		if name == alias {
			imports = append(imports, name)
		} else {
			imports = append(imports, fmt.Sprintf("%s as %s", name, alias))
		}
	}
	return strings.Join(imports, ", ")
}

// normalizeModulePath normalizes module paths to handle equivalent imports
func (f *Flattener) normalizeModulePath(modulePath string) string {
	// Handle Node.js built-in modules with node: prefix
	// node:fs/promises -> fs/promises
	// node:fs -> fs
	if strings.HasPrefix(modulePath, "node:") {
		return modulePath[5:] // Remove "node:" prefix
	}
	return modulePath
}

// resolveNamingConflicts checks for naming conflicts with imports from other modules
func (f *Flattener) resolveNamingConflicts(newDetails *ImportDetails) {
	// Check for conflicts with imports from other modules
	for modulePath, existingDetails := range f.externalImports {
		if modulePath == newDetails.ModulePath {
			continue // Skip same module (will be merged later)
		}

		// Check default import conflicts
		if newDetails.DefaultImport != "" && existingDetails.DefaultImport != "" &&
			newDetails.DefaultImport == existingDetails.DefaultImport {
			// Conflict: same default import name from different modules
			// Create alias for the new import
			newDetails.DefaultImport = f.createModuleAlias(newDetails.DefaultImport, newDetails.ModulePath)
		}

		// Check namespace import conflicts
		if newDetails.NamespaceAlias != "" && existingDetails.NamespaceAlias != "" &&
			newDetails.NamespaceAlias == existingDetails.NamespaceAlias {
			// Conflict: same namespace alias from different modules
			newDetails.NamespaceAlias = f.createModuleAlias(newDetails.NamespaceAlias, newDetails.ModulePath)
		}

		// Check named import conflicts
		for name, alias := range newDetails.NamedImports {
			for existingName, existingAlias := range existingDetails.NamedImports {
				if alias == existingAlias {
					// Conflict: same alias from different modules
					if name == existingName {
						// Same import name but different modules - create alias for new import
						newDetails.NamedImports[name] = f.createModuleAlias(alias, newDetails.ModulePath)
					} else {
						// Different import names but same alias - create unique alias for new import
						newDetails.NamedImports[name] = f.createModuleAlias(alias, newDetails.ModulePath)
					}
				}
			}
		}
	}
}

// createModuleAlias creates a unique alias based on module name
func (f *Flattener) createModuleAlias(baseName, modulePath string) string {
	// Extract a clean identifier from the module path
	moduleIdentifier := f.extractModuleIdentifier(modulePath)
	return fmt.Sprintf("%s_%s", baseName, moduleIdentifier)
}

// extractModuleIdentifier extracts a clean identifier from module path
func (f *Flattener) extractModuleIdentifier(modulePath string) string {
	// Handle various module path formats
	// fs/promises -> fs_promises
	// @nestjs/common -> nestjs_common
	// react -> react

	identifier := modulePath

	// Remove @ prefix
	if strings.HasPrefix(identifier, "@") {
		identifier = identifier[1:]
	}

	// Replace slashes and special characters with underscores
	identifier = strings.ReplaceAll(identifier, "/", "_")
	identifier = strings.ReplaceAll(identifier, "-", "_")
	identifier = strings.ReplaceAll(identifier, ".", "_")

	// Ensure it starts with a letter or underscore
	if len(identifier) > 0 && identifier[0] >= '0' && identifier[0] <= '9' {
		identifier = "_" + identifier
	}

	return identifier
}
