# Solc-Go

Golang bindings for the [Solidity compiler](https://github.com/ethereum/solidity).

Uses the Emscripten compiled Solidity found in the [solc-bin repository](https://github.com/ethereum/solc-bin).

#### Example usage

Example:

```go
package main

import (
    "github.com/nmvalera/solc-go"
)

func main() {
    compiler := Solc6_2_0()

    input := &solc.Input{
		Language: "Solidity",
		Sources:  map[string]solc.SourceIn{
            "One.sol": SourceIn{Content: "pragma solidity ^0.6.2; contract One { function one() public pure returns (uint) { return 1; } }"},
        },
		Settings: solc.Settings{
			Optimizer: solc.Optimizer{
				Enabled: true,
				Runs:    200,
			},
			EVMVersion: "byzantium",
			OutputSelection: map[string]map[string][]string{
				"*": map[string][]string{
					"*": []string{
						"abi",
						"evm.bytecode.object",
						"evm.bytecode.sourceMap",
						"evm.deployedBytecode.object",
						"evm.deployedBytecode.sourceMap",
						"evm.methodIdentifiers",
					},
					"": []string{
						"ast",
					},
				},
			},
		},
    }
    
    output, _ := compiler.Compile(input)

    fmt.Printf("Bytecode: %v", output.Contracts["One.sol"]["One"].EVM.Bytecode.Object)
}
```
