package solc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/fxamacker/cbor"
	"github.com/hashicorp/go-version"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
	"rogchap.com/v8go"
)

var (
	binList                 string
	releaseVersionListCache = make(map[string]string)
	compilerCache           = make(map[string]*Compiler)
	ver6, _                 = version.NewVersion("0.6.0")
	ver5, _                 = version.NewVersion("0.5.0")
)

type Compiler struct {
	isolate *v8go.Isolate
	ctx     *v8go.Context
	// protect underlying v8 context from concurrent access
	mux      *sync.Mutex
	compiler *v8go.Value
	ver      *version.Version
}

func NewCompiler(wasmScript, ver string) (*Compiler, error) {
	var err error

	c := &Compiler{
		mux: &sync.Mutex{},
	}

	c.ver, err = version.NewVersion(ver)
	if err != nil {
		return nil, err
	}

	c.isolate = v8go.NewIsolate()
	c.ctx = v8go.NewContext(c.isolate)

	if err = c.init(wasmScript); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Compiler) init(wasmScript string) error {
	var err error
	if _, err = c.ctx.RunScript(wasmScript, "main.js"); err != nil {
		return err
	}
	// less than version 0.5.0, use compileJSON
	// greater or equal 0.5.0 and less than 0.6.0, use solidity_compile('string', 'number')
	// greater or equal 0.6.0, use solidity_compile('string', 'number', 'number')
	if c.ver.LessThan(ver5) {
		c.compiler, err = c.ctx.RunScript("Module.cwrap('compileStandard', 'string', ['string', 'number'])",
			"wrap_compile.js")
	} else if c.ver.GreaterThanOrEqual(ver5) && c.ver.LessThan(ver6) {
		c.compiler, err = c.ctx.RunScript("Module.cwrap('solidity_compile', 'string', ['string', 'number', 'number'])",
			"wrap_compile.js")
	} else {
		c.compiler, err = c.ctx.RunScript("Module.cwrap('solidity_compile', 'string', ['string', 'number', 'number'])",
			"wrap_compile.js")
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Compiler) Compile(input *Input) (*Output, error) {
	fn, err := c.compiler.AsFunction()
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	value, err := v8go.NewValue(c.isolate, string(b))
	if err != nil {
		return nil, err
	}
	result, err := fn.Call(c.ctx.Global(), value)
	if err != nil {
		return nil, err
	}
	var output *Output
	if err = json.Unmarshal([]byte(result.String()), &output); err != nil {
		return nil, err
	}

	return output, nil
}

func (c *Compiler) Close() {
	c.mux.Lock()
	defer c.mux.Lock()
	c.ctx.Close()
	c.isolate.Dispose()
}

// ReleaseVersionList get release version list
func ReleaseVersionList() (map[string]string, error) {
	if len(releaseVersionListCache) == 0 {
		result := gjson.Get(binList, "releases")
		if err := json.Unmarshal([]byte(result.String()), &releaseVersionListCache); err != nil {
			return nil, err
		}
	}
	return releaseVersionListCache, nil
}

func GetCompiler(ver string) (*Compiler, error) {
	_, ok := compilerCache[ver]
	if ok {
		return compilerCache[ver], nil
	}

	var path string
	result := gjson.Get(binList, "builds")
	for _, item := range result.Array() {
		if item.Get("version").String() == ver {
			path = item.Get("path").String()
			break
		}
	}
	u := fmt.Sprintf("https://binaries.soliditylang.org/emscripten-wasm32/%s", path)
	resp := req.C().Get(u).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("get wasm error, status code:%d", resp.GetStatusCode())
	}

	wasmScript := resp.String()
	compiler, err := NewCompiler(wasmScript, ver)
	if err != nil {
		return nil, err
	}
	compilerCache[ver] = compiler
	return compiler, nil
}

// Verify verify bytecode is equal
// refer to https://docs.soliditylang.org/en/v0.8.17/metadata.html
func Verify(compiledByteCode, byteCode, byteCodeHash string) (bool, error) {
	// get hex from bytecodeHash
	b, err := cbor.Marshal(byteCodeHash, cbor.EncOptions{})
	if err != nil {
		return false, err
	}
	startStr := hex.EncodeToString(b)
	// get hex from "solc"
	b, err = cbor.Marshal("solc", cbor.EncOptions{})
	if err != nil {
		return false, err
	}
	endStr := hex.EncodeToString(b)
	reg := fmt.Sprintf(`\w+?(%s\w+)%s\w+?`, startStr, endStr)
	re := regexp.MustCompile(reg)
	fn := func(s string) string {
		rs := re.FindAllStringSubmatch(s, -1)
		if len(rs) > 0 && len(rs[0]) >= 2 {
			return strings.Replace(s, rs[0][1], "", -1)
		}
		return s
	}
	compiledByteCode = fn(compiledByteCode)
	byteCode = fn(byteCode)
	return compiledByteCode == byteCode, nil
}

func init() {
	if err := initSolcBinList(); err != nil {
		panic(err)
	}
}

func initSolcBinList() error {
	resp := req.C().Get("https://raw.githubusercontent.com/ethereum/solc-bin/gh-pages/emscripten-wasm32/list.json").Do()
	if resp.Err != nil {
		return resp.Err
	}
	if resp.IsError() {
		return fmt.Errorf("get bin list failed, status code:%d", resp.GetStatusCode())
	}
	binList = resp.String()
	return nil
}
