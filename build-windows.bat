@echo off
setlocal

set SCRIPT_DIR=%~dp0
set DIST_DIR=%SCRIPT_DIR%dist\windows\ffmpeg-gui-converter
set BIN_DIR=%DIST_DIR%\bin

if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

copy /Y "%SCRIPT_DIR%packaging\windows\README.txt" "%DIST_DIR%\README.txt" >nul

go build -trimpath -ldflags "-H windowsgui" -o "%DIST_DIR%\ffmpeg-gui-converter.exe" .
if errorlevel 1 exit /b 1

if exist "%SCRIPT_DIR%third_party\windows\bin\ffmpeg.exe" copy /Y "%SCRIPT_DIR%third_party\windows\bin\ffmpeg.exe" "%BIN_DIR%\ffmpeg.exe" >nul
if exist "%SCRIPT_DIR%bin\ffmpeg.exe" copy /Y "%SCRIPT_DIR%bin\ffmpeg.exe" "%BIN_DIR%\ffmpeg.exe" >nul
if exist "%SCRIPT_DIR%ffmpeg.exe" copy /Y "%SCRIPT_DIR%ffmpeg.exe" "%BIN_DIR%\ffmpeg.exe" >nul

if exist "%SCRIPT_DIR%third_party\windows\bin\ffprobe.exe" copy /Y "%SCRIPT_DIR%third_party\windows\bin\ffprobe.exe" "%BIN_DIR%\ffprobe.exe" >nul
if exist "%SCRIPT_DIR%bin\ffprobe.exe" copy /Y "%SCRIPT_DIR%bin\ffprobe.exe" "%BIN_DIR%\ffprobe.exe" >nul
if exist "%SCRIPT_DIR%ffprobe.exe" copy /Y "%SCRIPT_DIR%ffprobe.exe" "%BIN_DIR%\ffprobe.exe" >nul

echo.
echo Built Windows GUI package:
echo   %DIST_DIR%

if exist "%BIN_DIR%\ffmpeg.exe" (
  echo Bundled ffmpeg.exe
) else (
  echo Note: add ffmpeg.exe to %BIN_DIR% before distribution.
)

if exist "%BIN_DIR%\ffprobe.exe" (
  echo Bundled ffprobe.exe
) else (
  echo Note: add ffprobe.exe to %BIN_DIR% before distribution.
)
