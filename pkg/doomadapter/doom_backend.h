#ifndef DOOM_BACKEND_H
#define DOOM_BACKEND_H

// Headless backend for doomgeneric: implements the DG_* porting interface
// with no display/audio/timing side effects of its own. The Go caller
// drives ticking and pacing, and reads DG_ScreenBuffer directly.

// Push a key event into the input queue, to be drained by DG_GetKey on the
// next call. `key` must be a doomkeys.h KEY_* code (or ASCII for letters).
void DG_PushKey(int pressed, unsigned char key);

#endif // DOOM_BACKEND_H
