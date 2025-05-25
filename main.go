package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ashrhmn/ts-flatten/flatten"
)

func main() {
	// Define command line flags
	var localPrefixes string
	var rootDir string
	flag.StringVar(&localPrefixes, "include", "", "Comma-separated list of import prefixes to treat as local (e.g., src,main)")
	flag.StringVar(&localPrefixes, "i", "", "Comma-separated list of import prefixes to treat as local (short form)")
	flag.StringVar(&rootDir, "root", "", "Root directory for resolving path-mapped imports (defaults to parent of entry file)")
	flag.StringVar(&rootDir, "r", "", "Root directory for resolving path-mapped imports (short form)")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <typescript-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlattens TypeScript files by recursively inlining local imports.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <typescript-file>  The TypeScript file to flatten\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s src/main.ts\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --include src,main src/main.ts\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --include src --root /path/to/project src/nested/file.ts\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i src -r ../project src/main.ts\n", os.Args[0])
	}

	// Parse command line flags
	flag.Parse()

	// Check if exactly one non-flag argument is provided
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	entryFile := flag.Arg(0)

	// Parse local prefixes
	var additionalPrefixes []string
	if localPrefixes != "" {
		prefixes := strings.Split(localPrefixes, ",")
		for _, prefix := range prefixes {
			trimmed := strings.TrimSpace(prefix)
			if trimmed != "" {
				additionalPrefixes = append(additionalPrefixes, trimmed)
			}
		}
	}

	// Create a new flattener instance with additional prefixes
	flattener := flatten.NewFlattenerWithPrefixes(additionalPrefixes)

	// Set custom root directory if provided
	if rootDir != "" {
		absRootDir, err := filepath.Abs(rootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid root directory %s: %v\n", rootDir, err)
			os.Exit(1)
		}
		flattener.SetRootDir(absRootDir)
	}

	// Flatten the TypeScript file
	result, err := flattener.FlattenFile(entryFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output the flattened result to stdout
	fmt.Print(result)
}
