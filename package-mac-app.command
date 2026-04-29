#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
APP_NAME="FFmpeg GUI Converter"
BINARY_NAME="ffmpeg-gui-converter"
DIST_DIR="$SCRIPT_DIR/dist/macos"
APP_DIR="$DIST_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
RESOURCE_BIN_DIR="$RESOURCES_DIR/bin"
ICON_SOURCE="$SCRIPT_DIR/assets/macos/app-icon-source.png"
ICON_PNG="$SCRIPT_DIR/assets/macos/AppIcon-1024.png"
ICON_ICNS="$SCRIPT_DIR/assets/macos/AppIcon.icns"

if [[ "$(go env GOOS)" != "darwin" ]]; then
  print -u2 -- "This script must run on macOS."
  exit 1
fi

if [[ ! -f "$ICON_SOURCE" ]]; then
  print -u2 -- "Missing icon source: $ICON_SOURCE"
  exit 1
fi

copy_optional_binary() {
  local binary_name="$1"
  local source=""

  if [[ -x "$SCRIPT_DIR/$binary_name" ]]; then
    source="$SCRIPT_DIR/$binary_name"
  elif [[ -x "$SCRIPT_DIR/bin/$binary_name" ]]; then
    source="$SCRIPT_DIR/bin/$binary_name"
  elif command -v "$binary_name" >/dev/null 2>&1; then
    source=$(command -v "$binary_name")
  fi

  if [[ -n "$source" ]]; then
    cp -f "$source" "$RESOURCE_BIN_DIR/$binary_name"
    chmod +x "$RESOURCE_BIN_DIR/$binary_name"
    print -r -- "Bundled $binary_name from $source"
  else
    print -r -- "Warning: $binary_name not found; the app will fall back to PATH at runtime."
  fi
}

rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCE_BIN_DIR"

python3 "$SCRIPT_DIR/packaging/macos/make_app_icon.py" \
  "$ICON_SOURCE" \
  "$ICON_PNG" \
  "$ICON_ICNS"

cd "$SCRIPT_DIR"
go build -trimpath -o "$MACOS_DIR/$BINARY_NAME" .
chmod +x "$MACOS_DIR/$BINARY_NAME"
cp "$SCRIPT_DIR/packaging/macos/Info.plist" "$CONTENTS_DIR/Info.plist"
cp "$ICON_ICNS" "$RESOURCES_DIR/AppIcon.icns"

copy_optional_binary "ffmpeg"
copy_optional_binary "ffprobe"

print -r -- ""
print -r -- "Built macOS app bundle:"
print -r -- "  $APP_DIR"
print -r -- ""
print -r -- "You can now drag it into Applications, or run ./install-mac-app.command"
