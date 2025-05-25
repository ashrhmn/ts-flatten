package flatten

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a temporary directory structure for testing
func createTestFiles(t *testing.T, files map[string]string) string {
	tempDir := t.TempDir()

	for relativePath, content := range files {
		fullPath := filepath.Join(tempDir, relativePath)

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Write the file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	return tempDir
}

func TestFlattenFile_NoImports(t *testing.T) {
	files := map[string]string{
		"main.ts": `console.log("Hello, world!");
const x = 42;
export default x;`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := `// File: main.ts
console.log("Hello, world!");
const x = 42;
export default x;`

	if strings.TrimSpace(result) != strings.TrimSpace(expected) {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestFlattenFile_SingleLocalImport(t *testing.T) {
	files := map[string]string{
		"main.ts": `import { helper } from './utils';
console.log(helper());`,
		"utils.ts": `export function helper() {
    return "Hello from helper!";
}`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain both files with proper headers
	if !strings.Contains(result, "// File: main.ts") {
		t.Error("Result should contain main.ts file header")
	}
	if !strings.Contains(result, "// File: utils.ts") {
		t.Error("Result should contain utils.ts file header")
	}
	if !strings.Contains(result, "export function helper()") {
		t.Error("Result should contain utils.ts content")
	}
	if !strings.Contains(result, "console.log(helper())") {
		t.Error("Result should contain main.ts content")
	}
	// Should not contain the import statement
	if strings.Contains(result, "import { helper } from './utils'") {
		t.Error("Result should not contain import statements for local imports")
	}
}

func TestFlattenFile_ExternalImports(t *testing.T) {
	files := map[string]string{
		"main.ts": `import React from 'react';
import { lodash } from 'lodash';
import * as fs from 'fs';
import './local';

console.log("Using external libraries");`,
		"local.ts": `export const localVar = "local";`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should keep external imports
	if !strings.Contains(result, "import React from 'react'") {
		t.Error("Should preserve React import")
	}
	if !strings.Contains(result, "import { lodash } from 'lodash'") {
		t.Error("Should preserve lodash import")
	}
	if !strings.Contains(result, "import * as fs from 'fs'") {
		t.Error("Should preserve fs import")
	}

	// Should inline local import
	if !strings.Contains(result, "// File: local.ts") {
		t.Error("Should inline local.ts file")
	}
	if !strings.Contains(result, "export const localVar") {
		t.Error("Should include local.ts content")
	}
	if strings.Contains(result, "import './local'") {
		t.Error("Should not contain local import statement")
	}
}

func TestFlattenFile_NestedImports(t *testing.T) {
	files := map[string]string{
		"main.ts": `import { Component } from './components/index';
console.log(Component);`,
		"components/index.ts": `import { Button } from './Button';
import { Input } from '../ui/Input';
export { Button, Input as Component };`,
		"components/Button.ts": `export const Button = "Button component";`,
		"ui/Input.ts":          `export const Input = "Input component";`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain all files
	expectedFiles := []string{
		"// File: main.ts",
		"// File: components/index.ts",
		"// File: components/Button.ts",
		"// File: ui/Input.ts",
	}

	for _, expected := range expectedFiles {
		if !strings.Contains(result, expected) {
			t.Errorf("Result should contain: %s", expected)
		}
	}

	// Should contain content from all files
	expectedContent := []string{
		"Button component",
		"Input component",
		"export { Button, Input as Component }",
		"console.log(Component)",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Result should contain: %s", expected)
		}
	}
}

func TestFlattenFile_CyclicImports(t *testing.T) {
	files := map[string]string{
		"a.ts": `import { b } from './b';
export const a = "from a";
console.log(b);`,
		"b.ts": `import { a } from './a';
export const b = "from b";
console.log(a);`,
	}

	tempDir := createTestFiles(t, files)
	aFile := filepath.Join(tempDir, "a.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(aFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain cycle detection message
	if !strings.Contains(result, "already processed - avoiding cycle") {
		t.Error("Should detect and handle cyclic imports")
	}

	// Should still contain both files initially
	if !strings.Contains(result, "// File: a.ts") {
		t.Error("Should contain a.ts file header")
	}
	if !strings.Contains(result, "// File: b.ts") {
		t.Error("Should contain b.ts file header")
	}
}

func TestFlattenFile_TypeScriptExtensions(t *testing.T) {
	files := map[string]string{
		"main.ts": `import Component from './Component';
console.log(Component);`,
		"Component.tsx": `export default function Component() {
    return <div>Hello</div>;
}`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(result, "// File: Component.tsx") {
		t.Error("Should resolve .tsx extension")
	}
	if !strings.Contains(result, "export default function Component") {
		t.Error("Should include TSX content")
	}
}

func TestFlattenFile_IndexFiles(t *testing.T) {
	files := map[string]string{
		"main.ts": `import utils from './utils';
console.log(utils);`,
		"utils/index.ts": `export default "utils index";`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(result, "// File: utils/index.ts") {
		t.Error("Should resolve index.ts files")
	}
	if !strings.Contains(result, "utils index") {
		t.Error("Should include index.ts content")
	}
}

func TestFlattenFile_MissingFile(t *testing.T) {
	files := map[string]string{
		"main.ts": `import { missing } from './missing';
console.log(missing);`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	_, err := flattener.FlattenFile(mainFile)

	if err == nil {
		t.Error("Expected error for missing file")
	}

	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected 'failed to read file' error, got: %v", err)
	}
}

func TestParseImports(t *testing.T) {
	content := `import React from 'react';
import { useState, useEffect } from 'react';
import * as utils from './utils';
import './styles.css';
import type { User } from './types';
import type Config from './config';

const component = () => {
    return <div>Hello</div>;
};`

	flattener := NewFlattener()
	imports, contentWithoutImports := flattener.parseImports(content)

	expectedImports := []string{
		"react",
		"react",
		"./utils",
		"./styles.css",
		"./types",
		"./config",
	}

	if len(imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d", len(expectedImports), len(imports))
	}

	for i, expected := range expectedImports {
		if i < len(imports) && imports[i].Path != expected {
			t.Errorf("Expected import path %s, got %s", expected, imports[i].Path)
		}
	}

	// Content should not contain import statements
	if strings.Contains(contentWithoutImports, "import React") {
		t.Error("Content should not contain import statements")
	}

	// Content should still contain the component
	if !strings.Contains(contentWithoutImports, "const component = ()") {
		t.Error("Content should still contain non-import code")
	}
}

func TestIsLocalImport(t *testing.T) {
	flattener := NewFlattener()

	testCases := []struct {
		path     string
		expected bool
	}{
		{"./utils", true},
		{"../components", true},
		{".", true},
		{"..", true},
		{"react", false},
		{"@types/react", false},
		{"lodash", false},
		{"node_modules/react", false},
	}

	for _, tc := range testCases {
		result := flattener.isLocalImport(tc.path)
		if result != tc.expected {
			t.Errorf("For path %s, expected %v, got %v", tc.path, tc.expected, result)
		}
	}
}

func TestFlattenFile_DuplicateExternalImports(t *testing.T) {
	files := map[string]string{
		"main.ts": `import React from 'react';
import { useState } from 'react';
import lodash from 'lodash';
import './helper1';
import './helper2';

console.log('main');`,
		"helper1.ts": `import React from 'react';
import { useState } from 'react';
import axios from 'axios';

export const helper1 = 'helper1';`,
		"helper2.ts": `import React from 'react';
import lodash from 'lodash';
import { format } from 'date-fns';

export const helper2 = 'helper2';`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have external imports at the top
	lines := strings.Split(result, "\n")
	if len(lines) < 5 {
		t.Fatal("Result should have multiple lines")
	}

	// First several lines should be imports
	importCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "import ") {
			importCount++
		} else if strings.TrimSpace(line) == "" {
			// Empty line after imports is OK
			continue
		} else if strings.HasPrefix(line, "//") {
			// Found file header, imports section is done
			break
		}
	}

	// Should have exactly 5 unique external imports
	expectedImports := []string{
		"import React from 'react'",
		"import { useState } from 'react'",
		"import lodash from 'lodash'",
		"import axios from 'axios'",
		"import { format } from 'date-fns'",
	}

	if importCount != len(expectedImports) {
		t.Errorf("Expected %d external imports, got %d", len(expectedImports), importCount)
	}

	// Check that each expected import appears exactly once
	for _, expectedImport := range expectedImports {
		count := strings.Count(result, expectedImport)
		if count != 1 {
			t.Errorf("Expected import '%s' to appear exactly once, but found %d occurrences", expectedImport, count)
		}
	}

	// Should contain file headers for local imports
	if !strings.Contains(result, "// File: helper1.ts") {
		t.Error("Should contain helper1.ts file header")
	}
	if !strings.Contains(result, "// File: helper2.ts") {
		t.Error("Should contain helper2.ts file header")
	}
}
