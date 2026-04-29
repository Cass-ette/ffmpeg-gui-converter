@echo off
set SCRIPT_DIR=%~dp0
if exist "%SCRIPT_DIR%bin\ffmpeg-gui-converter.exe" (
  "%SCRIPT_DIR%bin\ffmpeg-gui-converter.exe"
) else (
  go run .
)
