package solc

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// importResolver handles the recursive resolution of Solidity imports
type importResolver struct {
	importCallback  ImportCallback
	resolvedSources map[string]bool // tracks resolved imports to avoid cycles
	contextStack    []string        // current import context for relative path resolution
	maxDepth        int             // maximum recursion depth
}

// newImportResolver creates a new import resolver
func newImportResolver(callback ImportCallback) *importResolver {
	return &importResolver{
		importCallback:  callback,
		resolvedSources: make(map[string]bool),
		contextStack:    []string{},
		maxDepth:        50,
	}
}

// resolveImports recursively resolves all imports in the input
func (r *importResolver) resolveImports(input *Input) (*Input, error) {
	if input.Sources == nil {
		input.Sources = make(map[string]SourceIn)
	}

	// Recursively resolve imports for each source file
	for fileName := range input.Sources {
		if err := r.resolveFileImports(input, fileName, 0); err != nil {
			return nil, err
		}
	}

	return input, nil
}

// resolveFileImports resolves imports for a specific file
func (r *importResolver) resolveFileImports(input *Input, fileName string, depth int) error {
	if depth > r.maxDepth {
		return fmt.Errorf("maximum import depth exceeded for file: %s", fileName)
	}

	if r.resolvedSources[fileName] {
		return nil // Already resolved
	}

	source, exists := input.Sources[fileName]
	if !exists {
		return fmt.Errorf("source file not found: %s", fileName)
	}

	// Mark as resolved to avoid cycles
	r.resolvedSources[fileName] = true

	// Push current file to context stack for relative path resolution
	r.contextStack = append(r.contextStack, fileName)
	defer func() {
		if len(r.contextStack) > 0 {
			r.contextStack = r.contextStack[:len(r.contextStack)-1]
		}
	}()

	// Find all import statements
	imports, err := r.extractImports(source.Content)
	if err != nil {
		return fmt.Errorf("failed to extract imports from %s: %w", fileName, err)
	}

	// Resolve each import
	for _, importPath := range imports {
		resolvedPath := r.resolveAbsolutePath(importPath, fileName)

		// Skip if already in sources
		if _, exists := input.Sources[resolvedPath]; exists {
			// Still need to recursively resolve this file's imports
			if err := r.resolveFileImports(input, resolvedPath, depth+1); err != nil {
				return err
			}
			continue
		}

		// Call the import callback to get the content
		result := r.importCallback(resolvedPath)
		if result.Error != "" {
			return fmt.Errorf("import resolution failed for %s: %s", resolvedPath, result.Error)
		}

		// Add the resolved source to input
		input.Sources[resolvedPath] = SourceIn{Content: result.Contents}

		// Recursively resolve imports in the newly added file
		if err := r.resolveFileImports(input, resolvedPath, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// extractImports finds all import statements in Solidity source code
func (r *importResolver) extractImports(sourceCode string) ([]string, error) {
	// Regex pattern to match Solidity import statements
	// Matches: import "path"; import {symbol} from "path"; import * as name from "path";
	pattern := `import\s+(?:(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)\s+from\s+)?["']([^"']+)["']`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var imports []string
	matches := re.FindAllStringSubmatch(sourceCode, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}

	return imports, nil
}

// resolveAbsolutePath converts a relative import path to an absolute path
func (r *importResolver) resolveAbsolutePath(importPath, currentFile string) string {
	// If it's already absolute (doesn't start with . or ..), return as-is
	if !strings.HasPrefix(importPath, ".") {
		return importPath
	}

	// Get the directory of the current file
	currentDir := filepath.Dir(currentFile)

	// Resolve the relative path
	resolvedPath := filepath.Join(currentDir, importPath)

	// Clean the path to resolve .. and . components
	return filepath.Clean(resolvedPath)
}
