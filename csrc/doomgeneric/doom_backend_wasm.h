#ifndef DOOM_BACKEND_H
#define DOOM_BACKEND_H

// wasm/WASI backend for doomgeneric: implements the DG_* porting interface
// with no display/audio side effects of its own, plus a few extra exports
// the Go/wazero host uses to drive the game and read frames.

// Push a key event into the input queue, to be drained by DG_GetKey on the
// next call. `key` must be a doomkeys.h KEY_* code (or ASCII for letters).
// Exported to the host as "DG_PushKey".
void DG_PushKey(int pressed, unsigned char key);

// Returns the address of DG_ScreenBuffer within this module's linear
// memory, so the host can read the framebuffer out directly.
// Exported to the host as "DG_GetScreenBuffer".
void* DG_GetScreenBuffer(void);

// Thin exported malloc/free so the host can write argv strings (e.g. the
// IWAD path) into this module's linear memory before calling
// doomgeneric_Create. Exported as "doomadapter_alloc"/"doomadapter_free".
void* doomadapter_alloc(int size);
void doomadapter_free(void* ptr);

#endif // DOOM_BACKEND_H
