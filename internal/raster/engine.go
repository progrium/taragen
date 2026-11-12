// Package raster turns SVG into PNG with resvg, compiled to WebAssembly and
// run by wazero, so taragen stays a pure-Go binary.
//
// The module's source is in module/. Rebuild resvg.wasm with `make resvg`.
package raster

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed resvg.wasm
var moduleWasm []byte

// memoryLimitPages caps each instance's linear memory at 1 GiB (64 KiB
// pages). A 4096 by 4096 card at scale 2 needs about 512 MiB.
const memoryLimitPages = 16384

var (
	engineOnce sync.Once
	engineRT   wazero.Runtime
	engineMod  wazero.CompiledModule
	engineErr  error

	// compiles counts module compilations, for tests.
	compiles int
)

// compiled returns the runtime and compiled module, compiling on first use
// so sites without raster outputs never pay for it.
func compiled() (wazero.Runtime, wazero.CompiledModule, error) {
	engineOnce.Do(func() {
		ctx := context.Background()
		cfg := wazero.NewRuntimeConfig().
			WithMemoryLimitPages(memoryLimitPages).
			WithCloseOnContextDone(true)
		engineRT = wazero.NewRuntimeWithConfig(ctx, cfg)
		engineMod, engineErr = engineRT.CompileModule(ctx, moduleWasm)
		compiles++
	})
	return engineRT, engineMod, engineErr
}

// Compiled reports whether the module has been compiled in this process.
func Compiled() bool {
	return compiles > 0
}

// instance is one instantiation of the module. It's not safe for concurrent
// use.
type instance struct {
	mod     api.Module
	alloc   api.Function
	dealloc api.Function
}

func newInstance(ctx context.Context) (*instance, error) {
	rt, cm, err := compiled()
	if err != nil {
		return nil, fmt.Errorf("compiling resvg module: %w", err)
	}
	// No WASI and no host functions: the module imports nothing, so it can
	// reach no filesystem, network, clock, or environment.
	mod, err := rt.InstantiateModule(ctx, cm, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, fmt.Errorf("instantiating resvg module: %w", err)
	}
	return &instance{
		mod:     mod,
		alloc:   mod.ExportedFunction("alloc"),
		dealloc: mod.ExportedFunction("dealloc"),
	}, nil
}

func (in *instance) close(ctx context.Context) {
	in.mod.Close(ctx)
}

// errTrap marks a failure inside the module itself, usually running out of
// memory, after which the instance can't be used again.
var errTrap = errors.New("resvg module trapped")

// call writes req into module memory, calls fn with it, and returns the
// response body.
func (in *instance) call(ctx context.Context, fn string, req []byte) ([]byte, error) {
	res, err := in.alloc.Call(ctx, uint64(len(req)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errTrap, err)
	}
	ptr := uint32(res[0])
	if !in.mod.Memory().Write(ptr, req) {
		return nil, fmt.Errorf("%w: request out of range", errTrap)
	}
	res, err = in.mod.ExportedFunction(fn).Call(ctx, uint64(ptr), uint64(len(req)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errTrap, err)
	}
	out := uint32(res[0])
	head, ok := in.mod.Memory().Read(out, 8)
	if !ok {
		return nil, fmt.Errorf("%w: response out of range", errTrap)
	}
	status := binary.LittleEndian.Uint32(head)
	n := binary.LittleEndian.Uint32(head[4:])
	body, ok := in.mod.Memory().Read(out+8, n)
	if !ok {
		return nil, fmt.Errorf("%w: response out of range", errTrap)
	}
	body = append([]byte(nil), body...)
	if _, err := in.dealloc.Call(ctx, uint64(out), uint64(8+n)); err != nil {
		return nil, fmt.Errorf("%w: %v", errTrap, err)
	}
	if status != 0 {
		return nil, errors.New(string(body))
	}
	return body, nil
}

// request builds a request buffer in the module's encoding.
type request []byte

func (r request) u32(v uint32) request {
	return binary.LittleEndian.AppendUint32(r, v)
}

func (r request) blob(b []byte) request {
	return append(r.u32(uint32(len(b))), b...)
}

func (r request) str(s string) request {
	return append(r.u32(uint32(len(s))), s...)
}
