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

	// Should keep external imports (may be reformatted due to merging)
	if !strings.Contains(result, `from "react"`) {
		t.Error("Should preserve React import")
	}
	if !strings.Contains(result, `from "lodash"`) {
		t.Error("Should preserve lodash import")
	}
	if !strings.Contains(result, `from "fs"`) {
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

	// Should have 4 external imports (react imports merged, others separate)
	expectedModules := []string{
		`from "react"`,
		`from "lodash"`,
		`from "axios"`,
		`from "date-fns"`,
	}

	if importCount != len(expectedModules) {
		t.Errorf("Expected %d external imports, got %d", len(expectedModules), importCount)
	}

	// Check that each expected module appears exactly once
	for _, expectedModule := range expectedModules {
		count := strings.Count(result, expectedModule)
		if count != 1 {
			t.Errorf("Expected module '%s' to appear exactly once, but found %d occurrences", expectedModule, count)
		}
	}

	// Check that React imports are properly merged
	if !strings.Contains(result, "React") {
		t.Error("Should contain React import")
	}
	if !strings.Contains(result, "useState") {
		t.Error("Should contain useState import")
	}

	// Check that React imports are merged into a single statement
	reactImportCount := strings.Count(result, `from "react"`)
	if reactImportCount != 1 {
		t.Errorf("Expected exactly one import from react, but found %d", reactImportCount)
	}

	// Should contain file headers for local imports
	if !strings.Contains(result, "// File: helper1.ts") {
		t.Error("Should contain helper1.ts file header")
	}
	if !strings.Contains(result, "// File: helper2.ts") {
		t.Error("Should contain helper2.ts file header")
	}
}

func TestFlattenFile_MergeImportsFromSameModule(t *testing.T) {
	files := map[string]string{
		"main.ts": `import { Injectable } from "@nestjs/common";
import './helper1';
import './helper2';

export class MainService {}`,
		"helper1.ts": `import { Injectable } from "@nestjs/common";

export const helper1 = Injectable;`,
		"helper2.ts": `import { BadRequestException, Injectable } from "@nestjs/common";

export const helper2 = { BadRequestException, Injectable };`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	flattener := NewFlattener()
	result, err := flattener.FlattenFile(mainFile)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have only one import from @nestjs/common
	nestjsImportCount := strings.Count(result, `from "@nestjs/common"`)
	if nestjsImportCount != 1 {
		t.Errorf("Expected exactly one import from @nestjs/common, but found %d", nestjsImportCount)
	}

	// Should contain both Injectable and BadRequestException in the merged import
	if !strings.Contains(result, "Injectable") {
		t.Error("Should contain Injectable in merged import")
	}
	if !strings.Contains(result, "BadRequestException") {
		t.Error("Should contain BadRequestException in merged import")
	}

	// The merged import should contain both items
	if !strings.Contains(result, "BadRequestException, Injectable") && !strings.Contains(result, "Injectable, BadRequestException") {
		t.Error("Should merge Injectable and BadRequestException into single import statement")
	}

	// Injectable should not appear multiple times in import statements
	lines := strings.Split(result, "\n")
	injectableInImportCount := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") && strings.Contains(line, "Injectable") {
			injectableInImportCount++
		}
	}

	if injectableInImportCount != 1 {
		t.Errorf("Injectable should appear in exactly one import statement, but found %d", injectableInImportCount)
	}
}

func TestFlattenFile_AdditionalLocalPrefixes(t *testing.T) {
	files := map[string]string{
		"main.ts": `import React from 'react';
import { Button } from 'src/components/Button';
import { helper } from 'main/utils/helper';
import './local';

console.log('App with path mapping');`,
		"src/components/Button.tsx": `export const Button = () => {
    return "Button from src/components";
};`,
		"main/utils/helper.ts": `export function helper() {
    return "Helper from main/utils";
}`,
		"local.ts": `export const local = "Local file";`,
	}

	tempDir := createTestFiles(t, files)
	mainFile := filepath.Join(tempDir, "main.ts")

	// Test without additional prefixes - should treat src/ and main/ as external
	flattener1 := NewFlattener()
	result1, err := flattener1.FlattenFile(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain external imports for src and main
	if !strings.Contains(result1, `from "src/components/Button"`) {
		t.Error("Should preserve src/components/Button as external import")
	}
	if !strings.Contains(result1, `from "main/utils/helper"`) {
		t.Error("Should preserve main/utils/helper as external import")
	}
	// Should still inline local relative import
	if !strings.Contains(result1, "// File: local.ts") {
		t.Error("Should inline local relative import")
	}

	// Test with additional prefixes - should inline src/ and main/ imports
	flattener2 := NewFlattenerWithPrefixes([]string{"src", "main"})
	result2, err := flattener2.FlattenFile(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not contain external imports for src and main
	if strings.Contains(result2, `from "src/components/Button"`) {
		t.Error("Should inline src/components/Button, not treat as external")
	}
	if strings.Contains(result2, `from "main/utils/helper"`) {
		t.Error("Should inline main/utils/helper, not treat as external")
	}

	// Should contain file headers for inlined files
	if !strings.Contains(result2, "// File: src/components/Button.tsx") {
		t.Error("Should contain src/components/Button.tsx file header")
	}
	if !strings.Contains(result2, "// File: main/utils/helper.ts") {
		t.Error("Should contain main/utils/helper.ts file header")
	}
	if !strings.Contains(result2, "// File: local.ts") {
		t.Error("Should contain local.ts file header")
	}

	// Should contain content from inlined files
	if !strings.Contains(result2, "Button from src/components") {
		t.Error("Should contain content from src/components/Button.tsx")
	}
	if !strings.Contains(result2, "Helper from main/utils") {
		t.Error("Should contain content from main/utils/helper.ts")
	}

	// Should still preserve React import
	if !strings.Contains(result2, `from "react"`) {
		t.Error("Should preserve React as external import")
	}
}
