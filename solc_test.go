package solc

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type args struct {
	sources map[string]SourceIn
}

type res struct {
	errorsLen       int
	bytecode        map[string]map[string]Bytecode
	methodIdentiers map[string]map[string]map[string]string
	abisLen         map[string]map[string]int
}

type testCase struct {
	name      string
	commit    string
	args      args
	expectErr bool
	expectRes res
}

func TestSolc(t *testing.T) {
	tests := []testCase{
		// Solc 0.6.2 with pragma ^0.6.1
		{
			"Solc 0.6.2 with pragma ^0.6.1",
			"0.6.2+commit.bacdbe57",
			args{
				sources: map[string]SourceIn{
					"One.sol": SourceIn{Content: "pragma solidity ^0.6.1; contract One { function one() public pure returns (uint) { return 1; } }"},
				},
			},
			false,
			res{
				bytecode: map[string]map[string]Bytecode{
					"One.sol": map[string]Bytecode{
						"One": Bytecode{Object: "6080604052348015600f57600080fd5b50609c8061001e6000396000f3fe6080604052348015600f57600080fd5b50600436106044577c01000000000000000000000000000000000000000000000000000000006000350463901717d181146049575b600080fd5b604f6061565b60408051918252519081900360200190f35b60019056fea26469706673582212208c7c407543955dc2f62329d58792b557b7b6776ac58353f0d17e7ec75f2d3bfd64736f6c63430006020033"},
					},
				},
				abisLen: map[string]map[string]int{
					"One.sol": map[string]int{"One": 1},
				},
				methodIdentiers: map[string]map[string]map[string]string{
					"One.sol": map[string]map[string]string{
						"One": map[string]string{"one()": "901717d1"},
					},
				},
			},
		},
		// Solc 0.6.2 with pragma ^0.4.3
		{
			"Solc 0.6.2 with pragma ^0.4.3",
			"0.6.2+commit.bacdbe57",
			args{
				sources: map[string]SourceIn{
					"One.sol": SourceIn{Content: "pragma solidity ^0.4.3; contract One { function one() public pure returns (uint) { return 1; } }"},
				},
			},
			false,
			res{
				errorsLen: 1,
			},
		},
		// Solc 0.5.9 with pragma ^0.6.2 (Invalid)
		{
			"Solc 0.5.9 with pragma ^0.6.2",
			"0.5.9+commit.e560f70d",
			args{
				sources: map[string]SourceIn{
					"One.sol": SourceIn{Content: "pragma solidity ^0.6.2; contract One { function one() public pure returns (uint) { return 1; } }"},
				},
			},
			false,
			res{
				errorsLen: 1,
			},
		},
		// Solc 0.5.9 with pragma ^0.5.2
		{
			"Solc 0.5.9 with pragma ^0.5.2",
			"0.5.9+commit.e560f70d",
			args{
				sources: map[string]SourceIn{
					"One.sol": SourceIn{Content: "pragma solidity ^0.5.2; contract One { function one() public pure returns (uint) { return 1; } }"},
				},
			},
			false,
			res{
				bytecode: map[string]map[string]Bytecode{
					"One.sol": map[string]Bytecode{
						"One": Bytecode{Object: "6080604052348015600f57600080fd5b50609b8061001e6000396000f3fe6080604052348015600f57600080fd5b50600436106044577c01000000000000000000000000000000000000000000000000000000006000350463901717d181146049575b600080fd5b604f6061565b60408051918252519081900360200190f35b60019056fea265627a7a72305820690bfd951ab80f52d55fa4f9af420c83a8870e28e4913ed147d0aa31bd85c5db64736f6c63430005090032"},
					},
				},
				abisLen: map[string]map[string]int{
					"One.sol": map[string]int{"One": 1},
				},
				methodIdentiers: map[string]map[string]map[string]string{
					"One.sol": map[string]map[string]string{
						"One": map[string]string{"one()": "901717d1"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				testSolc(t, test)
			},
		)
	}
}

func testSolc(t *testing.T, test testCase) {
	// Read Solsjon file
	soljson, err := ioutil.ReadFile(fmt.Sprintf("./solc-bin/soljson-v%v.js", test.commit))
	require.NoError(t, err, "Soljson script should be loaded properly")

	// Create Solc object
	solc, err := New(string(soljson))
	require.NoError(t, err, "Creating Solc from valid solc emscripten binary should not error")
	assert.Greater(t, len(solc.License()), 10, "License should be valid")
	assert.Equal(t, fmt.Sprintf("%v.Emscripten.clang", test.commit), solc.Version(), "Version should be correct")

	// Prepare Compilation input
	in := &Input{
		Language: "Solidity",
		Sources:  test.args.sources,
		Settings: Settings{
			Optimizer: Optimizer{
				Enabled: true,
				Runs:    200,
			},
			EVMVersion: "byzantium",
			OutputSelection: map[string]map[string][]string{
				"*": map[string][]string{
					"*": []string{
						"abi",
						"devdoc",
						"userdoc",
						"metadata",
						"ir",
						"irOptimized",
						"storageLayout",
						"evm.bytecode.object",
						"evm.bytecode.sourceMap",
						"evm.bytecode.linkReferences",
						"evm.deployedBytecode.object",
						"evm.deployedBytecode.sourceMap",
						"evm.deployedBytecode.linkReferences",
						"evm.methodIdentifiers",
						"evm.gasEstimates",
					},
					"": []string{
						"ast",
						"legacyAST",
					},
				},
			},
		},
	}

	// Run compilation
	out, err := solc.Compile(in)
	if !test.expectErr {
		require.NoErrorf(t, err, "Compile should not error")
	} else {
		require.Errorf(t, err, "Compile should error")
	}

	// Test Errors
	require.Len(t, out.Errors, test.expectRes.errorsLen, "Invalid count of compilation error")

	// Test Bytecode
	for source, bytecodes := range test.expectRes.bytecode {
		for contract, bytecode := range bytecodes {
			assert.Equal(
				t,
				bytecode.Object,
				out.Contracts[source][contract].EVM.Bytecode.Object,
				"%v@%v: Bytecode does not match", contract, source,
			)
		}
	}

	// Test ABIs
	for source, abiLens := range test.expectRes.abisLen {
		for contract, abiLen := range abiLens {
			assert.Len(
				t,
				out.Contracts[source][contract].ABI,
				abiLen,
				"%v@%v: Incorrect ABI lenght", contract, source,
			)
		}
	}

	// Test method identifiers
	for source, contracts := range test.expectRes.methodIdentiers {
		for contract, methodIdentiers := range contracts {
			for method, methodIdentier := range methodIdentiers {
				assert.Equal(
					t,
					methodIdentier,
					out.Contracts[source][contract].EVM.MethodIdentifiers[method],
					"%v.%v@%v: Method identifier does not match", contract, method, source)
			}
		}
	}
}
