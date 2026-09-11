// Package doomadapter provides Go bindings for ozkl/doomgeneric, running
// the engine as a sandboxed WebAssembly module via wazero (no cgo).
// Each Game is a fully isolated instance (its own wasm linear memory), so
// multiple Games can run concurrently in one process.
//
// It exposes a headless engine instance: push key events in, pull
// rendered frames out. It owns no game loop, no I/O transport, and no
// window/audio system — the caller drives ticking and pacing and decides
// what to do with frames and input (e.g. streaming over a websocket).
package doomadapter

import (
	"context"
	_ "embed"
	"fmt"
	"image"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed build/doom.wasm
var wasmBinary []byte

// Width and Height are the fixed dimensions of the rendered framebuffer,
// matching doomgeneric's DOOMGENERIC_RESX/RESY compile-time defaults
// (build/doom.wasm was built with the defaults, 640x400).
const (
	Width  = 640
	Height = 400
)

var (
	sharedRuntime  wazero.Runtime
	compiledModule wazero.CompiledModule
	setupOnce      sync.Once
	setupErr       error
)

// setup compiles the embedded wasm module once and instantiates the WASI
// host module on a shared runtime; every Game gets its own isolated
// instantiation of compiledModule on this runtime (see New).
func setup(ctx context.Context) error {
	setupOnce.Do(func() {
		sharedRuntime = wazero.NewRuntime(ctx)

		if _, err := wasi_snapshot_preview1.Instantiate(ctx, sharedRuntime); err != nil {
			setupErr = fmt.Errorf("doomadapter: instantiate WASI: %w", err)
			return
		}

		compiledModule, setupErr = sharedRuntime.CompileModule(ctx, wasmBinary)
		if setupErr != nil {
			setupErr = fmt.Errorf("doomadapter: compile wasm module: %w", setupErr)
		}
	})
	return setupErr
}

var instanceSeq atomic.Uint64

// Game is a single running doomgeneric instance, backed by its own wasm
// module instantiation. Call Close when done with it.
type Game struct {
	mod api.Module

	fnTick            api.Function
	fnPushKey         api.Function
	fnGetScreenBuffer api.Function
}

// New starts a doomgeneric instance loading the given IWAD file. Each call
// to New produces an independent, isolated instance.
func New(wadPath string) (*Game, error) {
	ctx := context.Background()

	if err := setup(ctx); err != nil {
		return nil, err
	}

	dir, file := filepath.Split(wadPath)
	if dir == "" {
		dir = "."
	}
	const guestWadDir = "/wad"

	name := fmt.Sprintf("doomadapter-%d", instanceSeq.Add(1))
	cfg := wazero.NewModuleConfig().
		WithName(name).
		WithSysWalltime().
		WithSysNanotime().
		WithFSConfig(wazero.NewFSConfig().WithReadOnlyDirMount(dir, guestWadDir))

	mod, err := sharedRuntime.InstantiateModule(ctx, compiledModule, cfg)
	if err != nil {
		return nil, fmt.Errorf("doomadapter: instantiate module: %w", err)
	}

	g := &Game{
		mod:               mod,
		fnTick:            mod.ExportedFunction("doomgeneric_Tick"),
		fnPushKey:         mod.ExportedFunction("DG_PushKey"),
		fnGetScreenBuffer: mod.ExportedFunction("DG_GetScreenBuffer"),
	}

	if err := g.create(ctx, guestWadDir+"/"+file); err != nil {
		mod.Close(ctx)
		return nil, err
	}

	return g, nil
}

// create builds a C-style argv (["doomadapter", "-iwad", guestWadPath]) in
// the module's own linear memory and calls doomgeneric_Create.
func (g *Game) create(ctx context.Context, guestWadPath string) error {
	alloc := g.mod.ExportedFunction("doomadapter_alloc")
	args := []string{"doomadapter", "-iwad", guestWadPath}

	argPtrs := make([]uint64, len(args))
	for i, a := range args {
		b := append([]byte(a), 0) // NUL-terminate for C
		res, err := alloc.Call(ctx, uint64(len(b)))
		if err != nil {
			return fmt.Errorf("doomadapter: alloc argv[%d]: %w", i, err)
		}
		ptr := uint32(res[0])
		if !g.mod.Memory().Write(ptr, b) {
			return fmt.Errorf("doomadapter: write argv[%d] out of bounds", i)
		}
		argPtrs[i] = uint64(ptr)
	}

	argvRes, err := alloc.Call(ctx, uint64(len(args)*4))
	if err != nil {
		return fmt.Errorf("doomadapter: alloc argv array: %w", err)
	}
	argv := uint32(argvRes[0])
	for i, p := range argPtrs {
		if !g.mod.Memory().WriteUint32Le(argv+uint32(i*4), uint32(p)) {
			return fmt.Errorf("doomadapter: write argv pointer[%d] out of bounds", i)
		}
	}

	create := g.mod.ExportedFunction("doomgeneric_Create")
	if _, err := create.Call(ctx, uint64(len(args)), uint64(argv)); err != nil {
		return fmt.Errorf("doomadapter: doomgeneric_Create: %w", err)
	}
	return nil
}

// Tick advances the game by one engine tic. The caller is responsible for
// calling this at the desired rate (doom's native rate is ~35Hz).
func (g *Game) Tick() error {
	_, err := g.fnTick.Call(context.Background())
	return err
}

// PushKey enqueues a key press/release event to be consumed on the next
// Tick. key is a doomkeys.h KEY_* code (see keys.go).
func (g *Game) PushKey(pressed bool, key KeyCode) error {
	p := uint64(0)
	if pressed {
		p = 1
	}
	_, err := g.fnPushKey.Call(context.Background(), p, uint64(key))
	return err
}

// Close releases the wasm module instance and its linear memory. The Game
// must not be used afterward.
func (g *Game) Close() error {
	return g.mod.Close(context.Background())
}

// screenBufferPtr returns the guest-memory address of DG_ScreenBuffer, and
// is factored out for the test in doom_test.go.
func (g *Game) screenBufferPtr(ctx context.Context) (uint32, error) {
	res, err := g.fnGetScreenBuffer.Call(ctx)
	if err != nil {
		return 0, err
	}
	return uint32(res[0]), nil
}

// Frame returns the current framebuffer as an *image.RGBA, read out of the
// module's linear memory. It allocates a fresh image each call; the caller
// owns the result.
func (g *Game) Frame() (*image.RGBA, error) {
	ctx := context.Background()

	ptr, err := g.screenBufferPtr(ctx)
	if err != nil {
		return nil, fmt.Errorf("doomadapter: DG_GetScreenBuffer: %w", err)
	}

	raw, ok := g.mod.Memory().Read(ptr, uint32(Width*Height*4))
	if !ok {
		return nil, fmt.Errorf("doomadapter: framebuffer read out of bounds")
	}

	img := image.NewRGBA(image.Rect(0, 0, Width, Height))

	// DG_ScreenBuffer is a Width*Height array of pixel_t (uint32), packed
	// by doomgeneric as 0x00RRGGBB little-endian (see i_video.c's
	// rgba8888 mode; the alpha byte is left zero and is not meaningful
	// here), so in memory each pixel is [B, G, R, 0].
	for i := 0; i < Width*Height; i++ {
		o := i * 4
		img.Pix[o+0] = raw[o+2] // R
		img.Pix[o+1] = raw[o+1] // G
		img.Pix[o+2] = raw[o+0] // B
		img.Pix[o+3] = 0xff     // A
	}

	return img, nil
}
