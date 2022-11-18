# solc-go

golang binding for solidity compiler

# usage 
```bash
go get github.com/imxyb/solc-go
```

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/imxyb/solc-go"
)

func main() {
	// get build list
	buildList, err := solc.GetBuildList()
	if err != nil {
		panic(err)
	}
	fmt.Println(buildList)

	// get 0.6.0 complier
	compiler, err := solc.GetCompiler("0.8.13")
	if err != nil {
		panic(err)
	}

	// refer to https://docs.soliditylang.org/en/v0.8.17/using-the-compiler.html#input-description
	input := &solc.Input{
		Language: "Solidity",
		Sources: map[string]solc.SourceIn{
			"Token.sol": solc.SourceIn{
				Content: `
// SPDX-License-Identifier: MIT
// compiler version must be greater than or equal to 0.8.13 and less than 0.9.0
pragma solidity ^0.8.13;

contract HelloWorld {
    string public greet;

    constructor(string memory g) {
        greet = g;
    }
}
`,
			},
		},
		Settings: solc.Settings{
			Optimizer: solc.Optimizer{
				Enabled: false,
				Runs:    200,
			},
			OutputSelection: map[string]map[string][]string{
				"*": map[string][]string{
					"*": []string{"*"},
				},
			},
		},
	}

	// refer to https://docs.soliditylang.org/en/v0.8.17/using-the-compiler.html#output-description
	output, err := compiler.Compile(input)
	if err != nil {
		panic(err)
	}

	// get deployed code and metadata
	metadata := output.Contracts["Token.sol"]["HelloWorld"].Metadata
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(metadata), &m); err != nil {
		panic(err)
	}
	compiledByteCode := output.Contracts["Token.sol"]["HelloWorld"].EVM.DeployedBytecode.Object

	// verify code
	remoteByteCode := "608060..."
	verify, err := solc.Verify(compiledByteCode, remoteByteCode, m["bytecodeHash"].(string))
	if err != nil {
		panic(err)
	}
	fmt.Println(verify)
}
```