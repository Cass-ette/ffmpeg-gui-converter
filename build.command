#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$SCRIPT_DIR/bin"
cd "$SCRIPT_DIR"
go build -trimpath -o "$SCRIPT_DIR/bin/ffmpeg-gui-converter" .
print -r -- "Built $SCRIPT_DIR/bin/ffmpeg-gui-converter"
