package solc

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"
	"sync"

	"rogchap.com/v8go"
)

type Solc interface {
	License() string
	Version() string
	Compile(input *Input) (*Output, error)
	Close()
}

type baseSolc struct {
	isolate *v8go.Isolate
	ctx     *v8go.Context

	// protect underlying v8 context from concurrent access
	mux *sync.Mutex

	version *v8go.Value
	license *v8go.Value
	compile *v8go.Value
}

// New creates a new Solc binding using the underlying soljonjs emscripten binary
func New(soljsonjs string) (Solc, error) {
	return new(soljsonjs)
}

func new(soljsonjs string) (*baseSolc, error) {
	// Create v8go JS execution context
	isolate, err := v8go.NewIsolate()
	if err != nil {
		return nil, err
	}
	ctx, _ := v8go.NewContext(isolate)

	// Create Solc object
	solc := &baseSolc{
		mux:     &sync.Mutex{},
		isolate: isolate,
		ctx:     ctx,
	}

	// Initialize solc
	err = solc.init(soljsonjs)
	if err != nil {
		return nil, err
	}

	return solc, nil
}

func (solc *baseSolc) init(soljsonjs string) error {
	// Execute solcjson.js script
	_, err := solc.ctx.RunScript(soljsonjs, "soljson.js")
	if err != nil {
		return err
	}

	// Bind version function
	if strings.Contains(soljsonjs, "_solidity_version") {
		solc.version, err = solc.ctx.RunScript("Module.cwrap('solidity_version', 'string', [])", "wrap_version.js")
		if err != nil {
			return err
		}
	} else {
		solc.version, err = solc.ctx.RunScript("Module.cwrap('version', 'string', [])", "wrap_version.js")
		if err != nil {
			return err
		}
	}

	// Bind license function
	if strings.Contains(soljsonjs, "_solidity_license") {
		solc.license, err = solc.ctx.RunScript("Module.cwrap('solidity_license', 'string', [])", "wrap_license.js")
		if err != nil {
			return err
		}
	} else if strings.Contains(soljsonjs, "_license") {
		solc.license, err = solc.ctx.RunScript("Module.cwrap('license', 'string', [])", "wrap_license.js")
		if err != nil {
			return err
		}
	}

	// Bind compile function
	solc.compile, err = solc.ctx.RunScript("Module.cwrap('solidity_compile', 'string', ['string', 'number', 'number'])", "wrap_compile.js")
	if err != nil {
		return err
	}

	return nil
}

func (solc *baseSolc) Close() {
	solc.mux.Lock()
	defer solc.mux.Lock()
	solc.ctx.Close()
	solc.isolate.Close()
}

func (solc *baseSolc) License() string {
	if solc.license != nil {
		solc.mux.Lock()
		defer solc.mux.Lock()
		val, _ := solc.license.Call(solc.ctx, nil)
		return val.String()
	}
	return ""
}

func (solc *baseSolc) Version() string {
	if solc.version != nil {
		solc.mux.Lock()
		defer solc.mux.Lock()
		val, _ := solc.version.Call(solc.ctx, nil)
		return val.String()
	}
	return ""
}

func (solc *baseSolc) Compile(input *Input) (*Output, error) {
	// Marshal Solc Compiler Input
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	// Run Compilation
	solc.mux.Lock()
	defer solc.mux.Unlock()

	val_in, err := solc.ctx.Create(string(b))
	if err != nil {
		return nil, err
	}
	val_one, _ := solc.ctx.Create(1)
	val_out, err := solc.compile.Call(solc.ctx, nil, val_in, val_one, val_one)
	if err != nil {
		return nil, err
	}

	out := &Output{}
	err = json.Unmarshal([]byte(val_out.String()), out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func NewFromFile(file string) (Solc, error) {
	soljson, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return New(string(soljson))
}

const SOLC_BIN_DIR = "./solc-bin"

func Solc6_2_0() Solc {
	solc, err := NewFromFile(path.Join(SOLC_BIN_DIR, "soljson-v0.6.2+commit.bacdbe57.js"))
	if err != nil {
		// This should never happend unless binaries are replaced
		panic(err)
	}
	return solc
}

func Solc5_9_0() Solc {
	solc, err := NewFromFile(path.Join(SOLC_BIN_DIR, "soljson-v0.5.9+commit.e560f70d.js"))
	if err != nil {
		// This should never happend unless binaries are replaced
		panic(err)
	}
	return solc
}
