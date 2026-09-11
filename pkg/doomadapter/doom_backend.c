// Headless backend for doomgeneric.
//
// This replaces the SDL/X11/etc backend files upstream ships. It has no
// display, audio, or sleep side effects: the embedding Go program drives
// the game loop (calling doomgeneric_Tick) and reads DG_ScreenBuffer
// directly after each tick, and pushes input via DG_PushKey.

#include <time.h>

#include "doomgeneric.h"
#include "doom_backend.h"

#define KEYQUEUE_SIZE 16

static unsigned short s_KeyQueue[KEYQUEUE_SIZE];
static unsigned int s_KeyQueueReadIndex = 0;
static unsigned int s_KeyQueueWriteIndex = 0;

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
