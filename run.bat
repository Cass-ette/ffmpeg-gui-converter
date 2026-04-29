@echo off
set SCRIPT_DIR=%~dp0
if exist "%SCRIPT_DIR%dist\windows\ffmpeg-gui-converter\ffmpeg-gui-converter.exe" (
  start "" "%SCRIPT_DIR%dist\windows\ffmpeg-gui-converter\ffmpeg-gui-converter.exe"
  exit /b 0
)
if exist "%SCRIPT_DIR%bin\ffmpeg-gui-converter.exe" (
  start "" "%SCRIPT_DIR%bin\ffmpeg-gui-converter.exe"
) else (
  where go >nul 2>nul
  if errorlevel 1 (
    echo 未找到已构建的 Windows GUI 可执行文件，也没有检测到 Go 环境。
    echo 可以先运行 build-windows.bat，或者直接双击 dist\windows\ffmpeg-gui-converter\ffmpeg-gui-converter.exe
    pause
    exit /b 1
  )
  go run .
)
