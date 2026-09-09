// Package doomadapter provides Go (cgo) bindings for ozkl/doomgeneric,
// exposing a headless Doom engine instance: push key events in, pull
// rendered frames out. It owns no game loop, no I/O transport, and no
// window/audio system — the caller drives ticking and pacing and decides
// what to do with frames and input (e.g. streaming over a websocket).
package doomadapter

/*
#include <stdlib.h>
#include "doomgeneric.h"
#include "doom_backend.h"
*/
import "C"

import (
	"image"
	"sync"
	"unsafe"
)

// Width and Height are the fixed dimensions of the rendered framebuffer,
// matching doomgeneric's DOOMGENERIC_RESX/RESY compile-time defaults.
const (
	Width  = int(C.DOOMGENERIC_RESX)
	Height = int(C.DOOMGENERIC_RESY)
)

// process-wide lock: doomgeneric keeps all engine state in C globals, so
// only one Game may run per process.
//
// ponytail: global lock, not per-instance state. doomgeneric wasn't written
// to be reentrant (globals throughout), so supporting multiple concurrent
// instances would need upstream changes, not just a Go-side fix.
var gameMu sync.Mutex
var gameActive bool

// Game is a single running doomgeneric instance.
type Game struct{}

// New starts a doomgeneric instance loading the given IWAD file. Only one
// Game may exist per process at a time.
func New(wadPath string) (*Game, error) {
	gameMu.Lock()
	defer gameMu.Unlock()
	if gameActive {
		return nil, errAlreadyRunning
	}

	cArgv := []*C.char{
		C.CString("doomadapter"),
		C.CString("-iwad"),
		C.CString(wadPath),
	}
	defer func() {
		for _, a := range cArgv {
			C.free(unsafe.Pointer(a))
		}
	}()

	argv := make([]*C.char, len(cArgv))
	copy(argv, cArgv)

	C.doomgeneric_Create(C.int(len(argv)), &argv[0])

	gameActive = true
	return &Game{}, nil
}

// Tick advances the game by one engine tic. The caller is responsible for
// calling this at the desired rate (doom's native rate is ~35Hz).
func (g *Game) Tick() {
	C.doomgeneric_Tick()
}

// PushKey enqueues a key press/release event to be consumed on the next
// Tick. key is a doomkeys.h KEY_* code (see keys.go).
func (g *Game) PushKey(pressed bool, key KeyCode) {
	p := C.int(0)
	if pressed {
		p = 1
	}
	C.DG_PushKey(p, C.uchar(key))
}

// getKey drains one event from the input queue (test hook around
// DG_GetKey; production code never needs to read keys back out).
func getKey() (pressed bool, key byte, ok bool) {
	var p C.int
	var k C.uchar
	if C.DG_GetKey(&p, &k) == 0 {
		return false, 0, false
	}
	return p != 0, byte(k), true
}

// Frame returns the current framebuffer as an *image.RGBA. It allocates a
// fresh image each call; the caller owns the result.
func (g *Game) Frame() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))

	// DG_ScreenBuffer is a Width*Height array of pixel_t (uint32), packed
	// by doomgeneric as 0x00RRGGBB (see i_video.c's rgba8888 mode; the
	// alpha byte is left zero and is not meaningful here).
	buf := unsafe.Slice((*uint32)(unsafe.Pointer(C.DG_ScreenBuffer)), Width*Height)

	for i, px := range buf {
		o := i * 4
		img.Pix[o+0] = byte(px >> 16) // R
		img.Pix[o+1] = byte(px >> 8)  // G
		img.Pix[o+2] = byte(px)       // B
		img.Pix[o+3] = 0xff           // A
	}

	return img
}
