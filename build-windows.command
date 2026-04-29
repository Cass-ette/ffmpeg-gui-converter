#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
DIST_DIR="$SCRIPT_DIR/dist/windows/ffmpeg-gui-converter"
OUTPUT_EXE="$DIST_DIR/ffmpeg-gui-converter.exe"

resolve_windows_cc() {
  if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    print -r -- "x86_64-w64-mingw32-gcc"
    return 0
  fi
  if command -v zig >/dev/null 2>&1; then
    print -r -- "zig cc -target x86_64-windows-gnu"
    return 0
  fi
  return 1
}

copy_optional_windows_binary() {
  local binary_name="$1"
  local source=""

  for candidate in \
    "$SCRIPT_DIR/$binary_name" \
    "$SCRIPT_DIR/bin/$binary_name" \
    "$SCRIPT_DIR/third_party/windows/$binary_name" \
    "$SCRIPT_DIR/third_party/windows/bin/$binary_name"
  do
    if [[ -f "$candidate" ]]; then
      source="$candidate"
      break
    fi
  done

  if [[ -n "$source" ]]; then
    cp -f "$source" "$DIST_DIR/bin/$binary_name"
    print -r -- "Bundled $binary_name from $source"
  else
    print -r -- "Note: $binary_name not found locally; add it to $DIST_DIR/bin before distribution."
  fi
}

if ! WINDOWS_CC=$(resolve_windows_cc); then
  print -u2 -- "Missing Windows cross-compiler."
  print -u2 -- "Install mingw-w64 or zig, or build on a Windows machine with build-windows.bat."
  exit 1
fi

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/bin"
cp "$SCRIPT_DIR/packaging/windows/README.txt" "$DIST_DIR/README.txt"

cd "$SCRIPT_DIR"
env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="$WINDOWS_CC" \
  go build -trimpath -ldflags="-H windowsgui" -o "$OUTPUT_EXE" .

copy_optional_windows_binary "ffmpeg.exe"
copy_optional_windows_binary "ffprobe.exe"

print -r -- ""
print -r -- "Built Windows GUI package:"
print -r -- "  $DIST_DIR"
