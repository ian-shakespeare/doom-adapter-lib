package doomadapter

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/tetratelabs/wazero"
)

func testInstanceSeq() string {
	return strconv.FormatUint(instanceSeq.Add(1), 10)
}

// rawInstance instantiates the wasm module directly, without calling
// doomgeneric_Create, so tests can exercise doom_backend_wasm.c's
// DG_PushKey/DG_GetKey ring buffer without needing a WAD file.
func rawInstance(t *testing.T) *Game {
	t.Helper()
	ctx := context.Background()
	if err := setup(ctx); err != nil {
		t.Fatalf("setup: %v", err)
	}

	name := t.Name() + "-" + testInstanceSeq()
	mod, err := sharedRuntime.InstantiateModule(ctx, compiledModule, wazero.NewModuleConfig().WithName(name))
	if err != nil {
		t.Fatalf("InstantiateModule: %v", err)
	}
	t.Cleanup(func() { mod.Close(ctx) })

	return &Game{
		mod:               mod,
		fnTick:            mod.ExportedFunction("doomgeneric_Tick"),
		fnPushKey:         mod.ExportedFunction("DG_PushKey"),
		fnGetScreenBuffer: mod.ExportedFunction("DG_GetScreenBuffer"),
	}
}

func testGetKey(t *testing.T, g *Game) (pressed bool, key byte, ok bool) {
	t.Helper()
	ctx := context.Background()
	mem := g.mod.Memory()

	// scratch space at a fixed low address for the two out-params.
	const pressedPtr, keyPtr uint32 = 16, 20

	res, err := g.mod.ExportedFunction("DG_GetKey").Call(ctx, uint64(pressedPtr), uint64(keyPtr))
	if err != nil {
		t.Fatalf("DG_GetKey: %v", err)
	}
	if res[0] == 0 {
		return false, 0, false
	}
	p, _ := mem.ReadUint32Le(pressedPtr)
	k, _ := mem.ReadByte(keyPtr)
	return p != 0, k, true
}

// TestKeyQueue exercises doom_backend_wasm.c's ring buffer (push/drain
// through real wasm calls), which needs no WAD file and covers the one
// piece of non-trivial logic that's actually new code in this repo; the
// rest is vendored upstream engine code.
func TestKeyQueue(t *testing.T) {
	g := rawInstance(t)

	if err := g.PushKey(true, KeyFire); err != nil {
		t.Fatalf("PushKey: %v", err)
	}
	if err := g.PushKey(false, KeyLeftArrow); err != nil {
		t.Fatalf("PushKey: %v", err)
	}

	pressed, key, ok := testGetKey(t, g)
	if !ok || !pressed || key != byte(KeyFire) {
		t.Fatalf("got pressed=%v key=%#x ok=%v, want pressed=true key=%#x ok=true", pressed, key, ok, byte(KeyFire))
	}

	pressed, key, ok = testGetKey(t, g)
	if !ok || pressed || key != byte(KeyLeftArrow) {
		t.Fatalf("got pressed=%v key=%#x ok=%v, want pressed=false key=%#x ok=true", pressed, key, ok, byte(KeyLeftArrow))
	}

	_, _, ok = testGetKey(t, g)
	if ok {
		t.Fatal("expected queue to be empty")
	}
}

// TestMultipleInstancesAreIsolated pushes different keys into two separate
// Games and checks they don't see each other's input — the whole point of
// moving to wasm/wazero.
func TestMultipleInstancesAreIsolated(t *testing.T) {
	a := rawInstance(t)
	b := rawInstance(t)

	if err := a.PushKey(true, KeyFire); err != nil {
		t.Fatalf("PushKey: %v", err)
	}

	if _, _, ok := testGetKey(t, b); ok {
		t.Fatal("instance b saw a key pushed to instance a")
	}
	if _, _, ok := testGetKey(t, a); !ok {
		t.Fatal("instance a did not see its own pushed key")
	}
}

// TestNewTickFrame is a full end-to-end smoke test: requires a real IWAD
// file, path supplied via DOOMADAPTER_TEST_WAD (skipped otherwise since no
// WAD is vendored in this repo).
func TestNewTickFrame(t *testing.T) {
	wad := os.Getenv("DOOMADAPTER_TEST_WAD")
	if wad == "" {
		t.Skip("set DOOMADAPTER_TEST_WAD to a doom .wad path to run this test")
	}

	g, err := New(wad)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer g.Close()

	for i := 0; i < 10; i++ {
		if err := g.Tick(); err != nil {
			t.Fatalf("Tick: %v", err)
		}
	}

	frame, err := g.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
		t.Fatalf("got frame size %dx%d, want %dx%d", frame.Bounds().Dx(), frame.Bounds().Dy(), Width, Height)
	}
}
