#!/bin/zsh
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$SCRIPT_DIR/bin"
cd "$SCRIPT_DIR"
exec go build -o "$SCRIPT_DIR/bin/ffmpeg-gui-converter" .
