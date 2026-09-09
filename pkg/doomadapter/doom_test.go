package doomadapter

import (
	"os"
	"testing"
)

func cGetKey(t *testing.T) (pressed bool, key byte) {
	t.Helper()
	pressed, key, ok := getKey()
	if !ok {
		t.Fatal("getKey: queue unexpectedly empty")
	}
	return pressed, key
}

// TestKeyQueue exercises the C ring buffer directly (push/drain via cgo),
// which needs no WAD file and covers the one piece of non-trivial logic
// that's actually new code in this repo (doom_backend.c's DG_GetKey/
// DG_PushKey); the rest is vendored upstream engine code.
func TestKeyQueue(t *testing.T) {
	g := &Game{}
	g.PushKey(true, KeyFire)
	g.PushKey(false, KeyLeftArrow)

	pressed, key := cGetKey(t)
	if !pressed || key != byte(KeyFire) {
		t.Fatalf("got pressed=%v key=%#x, want pressed=true key=%#x", pressed, key, byte(KeyFire))
	}

	pressed, key = cGetKey(t)
	if pressed || key != byte(KeyLeftArrow) {
		t.Fatalf("got pressed=%v key=%#x, want pressed=false key=%#x", pressed, key, byte(KeyLeftArrow))
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

	for i := 0; i < 10; i++ {
		g.Tick()
	}

	frame := g.Frame()
	if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
		t.Fatalf("got frame size %dx%d, want %dx%d", frame.Bounds().Dx(), frame.Bounds().Dy(), Width, Height)
	}
}
