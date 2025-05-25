# ts-flatten

A command-line tool that flattens TypeScript files by recursively inlining all local imports while preserving external imports.

## Features

- **Recursive flattening**: Follows local import chains through multiple files and directories
- **Selective inlining**: Only inlines local imports (paths starting with `.` or `..`), preserves external imports
- **Cycle detection**: Prevents infinite loops when encountering circular dependencies
- **Multi-format support**: Handles `.ts`, `.tsx`, `.js`, `.jsx` files and index files
- **Robust path resolution**: Automatically resolves extensions and index files
- **Clean output**: Adds file headers and removes import statements for inlined files

## Installation

```bash
git clone https://github.com/ashrhmn/ts-flatten.git
cd ts-flatten
go build -o ts-flatten .
```

## Usage

```bash
./ts-flatten <path-to-typescript-file>
```

### Examples

```bash
# Flatten a single TypeScript file
./ts-flatten src/main.ts

# Flatten a TypeScript React component
./ts-flatten components/App.tsx

# Output to a file
./ts-flatten src/main.ts > flattened.ts
```

## How it works

1. **Parse entry file**: Reads the specified TypeScript file and extracts all import statements
2. **Classify imports**: Distinguishes between local imports (starting with `.` or `..`) and external imports
3. **Recursive processing**: For each local import:
   - Resolves the file path (handles extensions and index files)
   - Recursively processes the imported file
   - Inlines the content with a file header comment
4. **Preserve external imports**: Keeps external library imports unchanged
5. **Cycle detection**: Tracks processed files to avoid infinite loops

## Example

Given this file structure:

```
src/
├── main.ts
├── utils.ts
└── components/
    ├── index.ts
    └── Button.ts
```

**src/main.ts:**

```typescript
import React from "react";
import { Button } from "./components";
import { helper } from "./utils";

console.log("App starting");
```

**src/utils.ts:**

```typescript
export function helper() {
  return "Hello from helper";
}
```

**src/components/index.ts:**

```typescript
import { Button } from "./Button";
export { Button };
```

**src/components/Button.ts:**

```typescript
export const Button = () => <button>Click me</button>;
```

**Command:**

```bash
./ts-flatten src/main.ts
```

**Output:**

```typescript
// File: main.ts
import React from "react";
// File: components/index.ts
// File: components/Button.ts
export const Button = () => <button>Click me</button>;

export { Button };

// File: utils.ts
export function helper() {
  return "Hello from helper";
}

console.log("App starting");
```

## Supported Import Patterns

The tool recognizes and handles these TypeScript import patterns:

```typescript
// Named imports
import { something } from "./path";

// Default imports
import something from "./path";

// Namespace imports
import * as something from "./path";

// Side-effect imports
import "./path";

// Type imports
import type { Something } from "./path";
import type Something from "./path";
```

## Path Resolution

The tool follows Node.js module resolution for local files:

1. **Exact path**: `./file.ts` → `./file.ts`
2. **Extension resolution**: `./file` → `./file.ts`, `./file.tsx`, etc.
3. **Index resolution**: `./directory` → `./directory/index.ts`, `./directory/index.tsx`, etc.

## Testing

Run the test suite:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

The test suite covers:

- Files with no imports
- Single and nested local imports
- External import preservation
- Cyclic import detection
- Different file extensions
- Index file resolution
- Error handling for missing files

## Error Handling

The tool provides clear error messages for common issues:

- **Missing files**: When an imported file cannot be found
- **Invalid arguments**: When incorrect command-line arguments are provided
- **File read errors**: When files cannot be read due to permissions or other issues

## Limitations

- **Regex-based parsing**: Uses regular expressions instead of a full TypeScript AST parser for simplicity
- **Comment preservation**: Comments within import statements may not be perfectly preserved
- **Complex imports**: Very complex or dynamically constructed import statements may not be handled

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code follows Go conventions: `go fmt ./...`
3. New features include tests
4. Documentation is updated as needed

## License

MIT License - see LICENSE file for details.
