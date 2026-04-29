#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
APP_BUNDLE="$SCRIPT_DIR/dist/macos/FFmpeg GUI Converter.app"

if [[ -d "$APP_BUNDLE" ]]; then
  open "$APP_BUNDLE"
  exit 0
fi

if [[ -x "$SCRIPT_DIR/bin/ffmpeg-gui-converter" ]]; then
  exec "$SCRIPT_DIR/bin/ffmpeg-gui-converter"
fi

cd "$SCRIPT_DIR"
exec go run .
