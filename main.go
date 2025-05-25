package main

import (
	"fmt"
	"os"

	"github.com/ashrhmn/ts-flatten/flatten"
)

func main() {
	// Check if exactly one argument is provided
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <typescript-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s src/abc.ts\n", os.Args[0])
		os.Exit(1)
	}

	entryFile := os.Args[1]

	// Create a new flattener instance
	flattener := flatten.NewFlattener()

	// Flatten the TypeScript file
	result, err := flattener.FlattenFile(entryFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output the flattened result to stdout
	fmt.Print(result)
}
