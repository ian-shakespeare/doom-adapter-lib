#!/usr/bin/env bash
# Rebuilds pkg/doomadapter/build/doom.wasm from csrc/doomgeneric.
#
# Requires zig (https://ziglang.org) on PATH for its bundled wasm32-wasi
# target + wasi-libc; no other toolchain needed.
#
# The maintainers don't expect to need this often (the vendored engine
# source isn't expected to change), but it's kept for GPL source-availability
# compliance and in case doom_backend_wasm.c ever needs a change.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

zig cc --target=wasm32-wasi -Os -mexec-model=reactor \
  -Wl,--export=doomgeneric_Create -Wl,--export=doomgeneric_Tick \
  -Wl,--export=W_GetNumForName \
  csrc/doomgeneric/*.c \
  -o pkg/doomadapter/build/doom.wasm

echo "wrote pkg/doomadapter/build/doom.wasm"
