#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
APP_NAME="FFmpeg GUI Converter"
TARGET_DIR="${1:-/Applications}"
SOURCE_APP="$SCRIPT_DIR/dist/macos/$APP_NAME.app"
TARGET_APP="$TARGET_DIR/$APP_NAME.app"

"$SCRIPT_DIR/package-mac-app.command"

mkdir -p "$TARGET_DIR"
rm -rf "$TARGET_APP"
ditto "$SOURCE_APP" "$TARGET_APP"

print -r -- "Installed $TARGET_APP"
print -r -- "Launch it from Applications, or run: open \"$TARGET_APP\""
