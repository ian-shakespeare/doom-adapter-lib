// wasm/WASI headless backend for doomgeneric. See doom_backend_wasm.h.
//
// This replaces the SDL/X11/etc backend files upstream ships, and the cgo
// headless backend used in earlier versions of this library. The
// embedding Go program (via wazero) drives the game loop (calling
// doomgeneric_Tick), reads the framebuffer via DG_GetScreenBuffer + wasm
// linear memory, and pushes input via DG_PushKey.
//
// DG_GetTicksMs uses plain clock_gettime: wasi-libc implements this via
// the WASI clock_time_get import, which wazero's WASI implementation
// provides, so no custom host import is needed for timing.

#include <stdlib.h>
#include <time.h>

#include "doomgeneric.h"
#include "doom_backend_wasm.h"

// wasi-libc has no system(): i_system.c uses it only to spawn a native
// "zenity" error dialog on crash, which is meaningless in a headless wasm
// module. Stub it out so the linker resolves the symbol.
int system(const char *command)
{
    (void)command;
    return -1;
}


#define KEYQUEUE_SIZE 16

static unsigned short s_KeyQueue[KEYQUEUE_SIZE];
static unsigned int s_KeyQueueReadIndex = 0;
static unsigned int s_KeyQueueWriteIndex = 0;

__attribute__((export_name("DG_PushKey")))
void DG_PushKey(int pressed, unsigned char key)
{
    unsigned short keyData = ((unsigned short)(pressed ? 1 : 0) << 8) | key;

    s_KeyQueue[s_KeyQueueWriteIndex] = keyData;
    s_KeyQueueWriteIndex++;
    s_KeyQueueWriteIndex %= KEYQUEUE_SIZE;
    // ponytail: fixed-size ring buffer, oldest events silently overwritten
    // if the caller floods input faster than Tick() drains it. Bump
    // KEYQUEUE_SIZE or add overflow tracking if that ever matters.
}

__attribute__((export_name("DG_GetScreenBuffer")))
void* DG_GetScreenBuffer(void)
{
    return DG_ScreenBuffer;
}

__attribute__((export_name("doomadapter_alloc")))
void* doomadapter_alloc(int size)
{
    return malloc((size_t)size);
}

__attribute__((export_name("doomadapter_free")))
void doomadapter_free(void* ptr)
{
    free(ptr);
}

void DG_Init(void)
{
}

void DG_DrawFrame(void)
{
}

void DG_SleepMs(uint32_t ms)
{
    (void)ms;
}

uint32_t DG_GetTicksMs(void)
{
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint32_t)(ts.tv_sec * 1000L + ts.tv_nsec / 1000000L);
}

__attribute__((export_name("DG_GetKey")))
int DG_GetKey(int* pressed, unsigned char* doomKey)
{
    if (s_KeyQueueReadIndex == s_KeyQueueWriteIndex)
    {
        return 0;
    }

    unsigned short keyData = s_KeyQueue[s_KeyQueueReadIndex];
    s_KeyQueueReadIndex++;
    s_KeyQueueReadIndex %= KEYQUEUE_SIZE;

    *pressed = keyData >> 8;
    *doomKey = keyData & 0xFF;
    return 1;
}

void DG_SetWindowTitle(const char * title)
{
    (void)title;
}
