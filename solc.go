package solc

import (
	"encoding/json"
	"strings"

	"rogchap.com/v8go"
)

type Solc struct {
	isolate *v8go.Isolate
	ctx     *v8go.Context

	version *v8go.Value
	license *v8go.Value
	compile *v8go.Value
}

func New(soljsonjs string) (*Solc, error) {
	// Create v8go JS execution context
	isolate, err := v8go.NewIsolate()
	if err != nil {
		return nil, err
	}
	ctx, _ := v8go.NewContext(isolate)

	// Create Solc object
	solc := &Solc{
		isolate: isolate,
		ctx:     ctx,
	}

	// In solcjson.js 0.4.9 "print" function is missing and leads to execution error
	// So we declare it
	// _, err = ctx.RunScript(`if (typeof print==="undefined") {print = null};`, "")
	// if err != nil {
	// 	panic(err)
	// }

	// Initialize solc
	err = solc.init(soljsonjs)
	if err != nil {
		return nil, err
	}

	return solc, nil
}

func (solc *Solc) init(soljsonjs string) error {
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

func (solc *Solc) Close() {
	solc.ctx.Close()
	solc.isolate.Close()
}

func (solc *Solc) License() string {
	if solc.license != nil {
		val, _ := solc.license.Call(solc.ctx, nil)
		return val.String()
	}
	return ""
}

func (solc *Solc) Version() string {
	val, _ := solc.version.Call(solc.ctx, nil)
	return val.String()
}

func (solc *Solc) Compile(input *Input) (*Output, error) {
	// Marshal Solc Compiler Input
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	// Execute Compilation
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
