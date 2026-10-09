# ts-flatten

A command-line tool that flattens TypeScript files by recursively inlining all local imports while preserving external imports.

## Features

- **Recursive flattening**: Follows local import chains through multiple files and directories
- **Selective inlining**: Only inlines local imports (paths starting with `.` or `..`), preserves external imports
- **Advanced import merging**: Automatically removes duplicate external imports and intelligently merges imports from the same module
- **Node.js module normalization**: Handles `node:` prefix imports (e.g., `node:fs/promises` → `fs/promises`) and merges them correctly
- **Cross-module conflict resolution**: Automatically resolves naming conflicts when the same name is imported from different modules using smart aliasing
- **Cycle detection**: Prevents infinite loops when encountering circular dependencies
- **Multi-format support**: Handles `.ts`, `.tsx`, `.js`, `.jsx`, `.d.ts` files and index files
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
./ts-flatten [options] <path-to-typescript-file>
```

### Options

- `--include <prefixes>` or `-i <prefixes>`: Comma-separated list of import prefixes to treat as local imports (e.g., `src,main`)
- `--root <directory>` or `-r <directory>`: Root directory for resolving path-mapped imports (defaults to parent of entry file)

### Examples

```bash
# Flatten a single TypeScript file
./ts-flatten src/main.ts

# Flatten with TypeScript path mappings (e.g., tsconfig paths like "src/*")
./ts-flatten --include src,main src/main.ts

# Short form of include flag
./ts-flatten -i src src/main.ts

# Multiple prefixes
./ts-flatten --include src,components,utils src/main.ts

# Specify custom root directory for path resolution
./ts-flatten --include src --root /path/to/project src/nested/file.ts

# Output to a file
./ts-flatten src/main.ts > flattened.ts
```

## Build and release

Run `make build` to build `bin/ts-flatten` and `make test` to run tests.
Override the output directory with `make build BIN_DIR=/path/to/bin`.

After committing and pushing the changes to release, run one of these commands
from a clean working tree:

```sh
make tag-patch  # v0.0.1 -> v0.0.2
make tag-minor  # v0.0.2 -> v0.1.0
make tag-major  # v0.1.0 -> v1.0.0
```

Each command fetches tags from `origin`, increments the highest stable
`vMAJOR.MINOR.PATCH` tag, and creates and pushes the new Git tag, matching
`gh-agent`. Without an existing stable tag, `make tag-patch` starts at `v0.0.1`.

## How it works

1. **Parse entry file**: Reads the specified TypeScript file and extracts all import statements
2. **Classify imports**: Distinguishes between local imports (starting with `.` or `..`) and external imports
3. **Recursive processing**: For each local import:
   - Resolves the file path (handles extensions and index files)
   - Recursively processes the imported file
   - Inlines the content with a file header comment
4. **Preserve external imports**: Keeps external library imports unchanged
5. **Cycle detection**: Tracks processed files to avoid infinite loops

## TypeScript Path Mapping Support

The `--include` flag allows you to specify additional import prefixes that should be treated as local imports. This is particularly useful when your TypeScript project uses path mapping in `tsconfig.json`.

For example, if your `tsconfig.json` has:

```json
{
  "compilerOptions": {
    "paths": {
      "src/*": ["./src/*"],
      "components/*": ["./src/components/*"]
    }
  }
}
```

You can use:

```bash
./ts-flatten --include src,components src/main.ts
```

This will treat imports like `import { Button } from "src/components/Button"` as local imports and inline them, even though they don't start with `./` or `../`.

**Without `--include`**: These imports are treated as external and preserved as import statements.
**With `--include`**: These imports are resolved relative to your project root and inlined.

### Custom Root Directory

When working with nested files, you may need to specify a custom root directory for path resolution using the `--root` flag:

```bash
# If your entry file is deeply nested but src/ should resolve from project root
./ts-flatten --include src --root ../my-project src/deeply/nested/file.ts
```

This is particularly useful when:

- Your entry file is in a subdirectory but path mappings are relative to the project root
- You want to process files from different locations with consistent path resolution

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

## Advanced Import Handling

### Import Merging and Deduplication

The tool automatically merges imports from the same module and removes duplicates:

**Input:**

```typescript
// main.ts
import React from "react";
import { useState } from "react";

// helper.ts (inlined)
import React from "react";
import { useEffect } from "react";
```

**Output:**

```typescript
import React, { useState, useEffect } from "react";

// ... inlined content
```

### Node.js Module Normalization

Handles `node:` prefix imports and merges them with equivalent imports:

**Input:**

```typescript
// main.ts
import { writeFile } from "node:fs/promises";

// helper.ts (inlined)
import { readFile } from "fs/promises";
```

**Output:**

```typescript
import { writeFile, readFile } from "fs/promises";

// ... inlined content
```

### Cross-Module Conflict Resolution

When the same name is imported from different modules, the tool automatically creates aliases:

**Input:**

```typescript
// main.ts
import { writeFile } from "fs";

// helper.ts (inlined)
import { writeFile } from "fs/promises";
```

**Output:**

```typescript
import { writeFile } from "fs";
import { writeFile as writeFile_fs_promises } from "fs/promises";

// ... inlined content
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

// Node.js modules with node: prefix
import { readFile } from "node:fs/promises";
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
