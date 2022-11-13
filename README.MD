# solc-go

golang binding for solidity compiler

# usage 
```bash
go get github.com/imxyb/solc-go
```

```go
package main

import (
	"fmt"

	"github.com/imxyb/solc-go"
)

func main() {
	list, err := solc.ReleaseVersionList()
	if err != nil {
		panic(err)
	}
	// get release list
	fmt.Println(list)

	// get 0.6.0 complier
	compiler, err := solc.GetCompiler("0.6.0")
	if err != nil {
		panic(err)
	}

	// refer to https://docs.soliditylang.org/en/v0.8.17/using-the-compiler.html#input-description
	input := &solc.Input{
		Language: "",
		Sources:  nil,
		Settings: solc.Settings{},
	}
	// refer to https://docs.soliditylang.org/en/v0.8.17/using-the-compiler.html#output-description
	output, err := compiler.Compile(input)
	if err != nil {
		panic(err)
	}
	// get deployed code
	fmt.Println(output.Contracts["<your solidity filename>"]["<your contract name>"].EVM.DeployedBytecode)
}
```