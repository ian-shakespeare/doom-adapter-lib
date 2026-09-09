package doomadapter

import "errors"

var errAlreadyRunning = errors.New("doomadapter: a Game is already running in this process (doomgeneric is not reentrant)")
