package solc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"rogchap.com/v8go"
)

// ImportResult represents the result of an import callback.
type ImportResult struct {
	// Contents holds the file contents if import was successful.
	Contents string `json:"contents,omitempty"`
	// Error holds the error message if import failed.
	Error string `json:"error,omitempty"`
}

// ImportCallback is a function that resolves import statements.
// It receives the import URL and returns the file contents or an error.
type ImportCallback func(url string) ImportResult

// CompileOptions holds additional options for compilation.
type CompileOptions struct {
	// ImportCallback handles import resolution.
	ImportCallback ImportCallback
}

// Solc represents a Solidity compiler interface.
type Solc interface {
	// License returns the license information of the compiler.
	License() string
	// Version returns the version information of the compiler.
	Version() string
	// CompileWithOptions compiles Solidity source code with additional options like import callbacks.
	// Pass nil for options to use default compilation without import callbacks.
	CompileWithOptions(input *Input, options *CompileOptions) (*Output, error)
	// Close releases all resources associated with the compiler instance.
	Close() error
}

// baseSolc implements the Solc interface using V8 JavaScript engine.
type baseSolc struct {
	isolate *v8go.Isolate
	ctx     *v8go.Context

	// mu protects the underlying v8 context from concurrent access
	mu sync.Mutex

	version *v8go.Function
	license *v8go.Function

	closed bool
}

// New creates a new Solc binding using the provided soljson.js emscripten binary.
func New(soljsonjs string) (Solc, error) {
	return newBaseSolc(soljsonjs)
}

// newBaseSolc creates and initializes a new baseSolc instance.
func newBaseSolc(soljsonjs string) (*baseSolc, error) {
	if soljsonjs == "" {
		return nil, fmt.Errorf("soljsonjs cannot be empty")
	}
	// Create v8go JS execution context
	isolate := v8go.NewIsolate()
	ctx := v8go.NewContext(isolate)

	// Create Solc object
	solc := &baseSolc{
		isolate: isolate,
		ctx:     ctx,
	}

	// Initialize solc
	if err := solc.init(soljsonjs); err != nil {
		solc.cleanup()
		return nil, fmt.Errorf("failed to initialize compiler: %w", err)
	}

	return solc, nil
}

// init initializes the Solidity compiler by executing the soljson.js script
// and binding the necessary functions.
func (s *baseSolc) init(soljsonjs string) error {
	// Execute soljson.js script
	if _, err := s.ctx.RunScript(soljsonjs, "soljson.js"); err != nil {
		return fmt.Errorf("failed to execute soljson.js: %w", err)
	}

	// Set up debug logging function
	debugLogFunc := v8go.NewFunctionTemplate(s.isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			if os.Getenv("SOLC_DEBUG") == "1" {
				fmt.Printf("JS DEBUG: %s\n", args[0].String())
			}
		}
		return v8go.Undefined(s.isolate)
	})

	debugLog := debugLogFunc.GetFunction(s.ctx)
	if err := s.ctx.Global().Set("debugLog", debugLog); err != nil {
		return fmt.Errorf("failed to set debug function: %w", err)
	}

	// Bind version function
	versionFunc := "version"
	if strings.Contains(soljsonjs, "_solidity_version") {
		versionFunc = "solidity_version"
	}
	var err error
	versionVal, err := s.ctx.RunScript(fmt.Sprintf("Module.cwrap('%s', 'string', [])", versionFunc), "wrap_version.js")
	if err != nil {
		return fmt.Errorf("failed to bind version function: %w", err)
	}
	s.version, err = versionVal.AsFunction()
	if err != nil {
		return fmt.Errorf("version binding is not a function: %w", err)
	}

	// Bind license function
	if strings.Contains(soljsonjs, "_solidity_license") {
		licenseVal, err := s.ctx.RunScript("Module.cwrap('solidity_license', 'string', [])", "wrap_license.js")
		if err != nil {
			return fmt.Errorf("failed to bind license function: %w", err)
		}
		s.license, err = licenseVal.AsFunction()
		if err != nil {
			return fmt.Errorf("license binding is not a function: %w", err)
		}
	} else if strings.Contains(soljsonjs, "_license") {
		licenseVal, err := s.ctx.RunScript("Module.cwrap('license', 'string', [])", "wrap_license.js")
		if err != nil {
			return fmt.Errorf("failed to bind license function: %w", err)
		}
		s.license, err = licenseVal.AsFunction()
		if err != nil {
			return fmt.Errorf("license binding is not a function: %w", err)
		}
	}

	// Simple wrapper for basic compilation
	setupScript := `
		// Create the core compile function binding
		var nativeCompile = Module.cwrap('solidity_compile', 'string', ['string']);
		
		// Create solc interface - Go will handle import resolution
		var solc = {
			compile: function(input) {
				return nativeCompile(input);
			}
		};
		
		globalThis.solc = solc;
		globalThis.compile = nativeCompile;
		
		solc;
	`

	_, err = s.ctx.RunScript(setupScript, "compile_wrapper.js")
	if err != nil {
		return fmt.Errorf("failed to create compile wrapper: %w", err)
	}

	// Validate that the setup worked by checking if solc is available
	solcVal, err := s.ctx.Global().Get("solc")
	if err != nil {
		return fmt.Errorf("solc object not created: %w", err)
	}

	if solcVal.IsUndefined() || solcVal.IsNull() {
		return fmt.Errorf("solc object is undefined or null")
	}

	return nil
}

// Close releases all resources associated with the compiler instance.
func (s *baseSolc) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.cleanup()
	s.closed = true
	return nil
}

// cleanup releases V8 resources without acquiring the mutex.
func (s *baseSolc) cleanup() {
	if s.ctx != nil {
		s.ctx.Close()
		s.ctx = nil
	}
	if s.isolate != nil {
		s.isolate.Dispose()
		s.isolate = nil
	}
}

// License returns the license information of the compiler.
func (s *baseSolc) License() string {
	if s.license == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ""
	}

	val, err := s.license.Call(v8go.Undefined(s.ctx.Isolate()))
	if err != nil {
		return ""
	}
	return val.String()
}

// Version returns the version information of the compiler.
func (s *baseSolc) Version() string {
	if s.version == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ""
	}

	val, err := s.version.Call(v8go.Undefined(s.ctx.Isolate()))
	if err != nil {
		return ""
	}
	return val.String()
}

// CompileWithOptions compiles Solidity source code with additional options like import callbacks.
func (s *baseSolc) CompileWithOptions(input *Input, options *CompileOptions) (*Output, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Marshal Solc Compiler Input
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Run Compilation
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("compiler has been closed")
	}

	// Resolve imports if callback is provided
	if options != nil && options.ImportCallback != nil {
		resolver := newImportResolver(options.ImportCallback)

		var err error
		input, err = resolver.resolveImports(input)
		if err != nil {
			return nil, fmt.Errorf("import resolution failed: %w", err)
		}

		// Re-marshal the updated input
		inputJSON, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal updated input: %w", err)
		}
	}

	// Get the compile function
	compileVal, err := s.ctx.Global().Get("compile")
	if err != nil {
		return nil, fmt.Errorf("compile function not available: %w", err)
	}

	compileFunc, err := compileVal.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("compile is not a function: %w", err)
	}

	// Create input value
	valInput, err := v8go.NewValue(s.ctx.Isolate(), string(inputJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create input value: %w", err)
	}

	// Execute compilation
	valOutput, err := compileFunc.Call(v8go.Undefined(s.ctx.Isolate()), valInput)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	output := &Output{}
	if err := json.Unmarshal([]byte(valOutput.String()), output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return output, nil
}
