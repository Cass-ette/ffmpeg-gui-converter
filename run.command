#!/bin/zsh
set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

if [[ -x "$SCRIPT_DIR/bin/ffmpeg-gui-converter" ]]; then
  exec "$SCRIPT_DIR/bin/ffmpeg-gui-converter"
fi

exec go run .
