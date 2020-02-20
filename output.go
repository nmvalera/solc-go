package solc

import (
	"encoding/json"
)

type Output struct {
	Errors    []Error                        `json:"errors,omitempty"`
	Sources   map[string]SourceOut           `json:"sources,omitempty"`
	Contracts map[string]map[string]Contract `json:"contracts,omitempty"`
}

type Error struct {
	SourceLocation   SourceLocation `json:"sourceLocation,omitempty"`
	Type             string         `json:"type,omitempty"`
	Component        string         `json:"component,omitempty"`
	Severity         string         `json:"severity,omitempty"`
	Message          string         `json:"message,omitempty"`
	FormattedMessage string         `json:"formattedMessage,omitempty"`
}

type SourceLocation struct {
	File  string `json:"file,omitempty"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

type SourceOut struct {
	ID        int             `json:"id,omitempty"`
	AST       json.RawMessage `json:"ast,omitempty"`
	LegacyAST json.RawMessage `json:"legacyAST,omitempty"`
}

type Contract struct {
	ABI      []json.RawMessage `json:"abi,omitempty"`
	Metadata string            `json:"metadata,omitempty"`
	UserDoc  json.RawMessage   `json:"userdoc,omitempty"`
	DevDoc   json.RawMessage   `json:"devdoc,omitempty"`
	IR       string            `json:"ir,omitempty"`
	// StorageLayout StorageLayout     `json:"storageLayout,omitempty"`
	EVM   EVM   `json:"evm,omitempty"`
	EWASM EWASM `json:"ewasm,omitempty"`
}

type EVM struct {
	Assembly          string                       `json:"assembly,omitempty"`
	LegacyAssembly    json.RawMessage              `json:"legacyAssembly,omitempty"`
	Bytecode          Bytecode                     `json:"bytecode,omitempty"`
	DeployedBytecode  Bytecode                     `json:"deployedBytecode,omitempty"`
	MethodIdentifiers map[string]string            `json:"methodIdentifiers,omitempty"`
	GasEstimates      map[string]map[string]string `json:"gasEstimates,omitempty"`
}

type Bytecode struct {
	Object         string                                `json:"object,omitempty"`
	Opcodes        string                                `json:"opcodes,omitempty"`
	SourceMap      string                                `json:"sourceMap,omitempty"`
	LinkReferences map[string]map[string][]LinkReference `json:"linkReferences,omitempty"`
}

type LinkReference struct {
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

type EWASM struct {
	Wast string `json:"wast,omitempty"`
	Wasm string `json:"wasm,omitempty"`
}
