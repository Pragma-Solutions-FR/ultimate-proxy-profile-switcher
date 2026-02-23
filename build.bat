@echo off
setlocal EnableDelayedExpansion

set BINARY=ultimate-proxy-profile-switcher
set LDFLAGS=-s -w

for /f "tokens=*" %%v in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%v
if "!VERSION!"=="" set VERSION=dev

if exist dist rmdir /s /q dist
mkdir dist

call :build linux   amd64
call :build linux   arm64
call :build darwin  amd64
call :build darwin  arm64
call :build windows amd64

echo.
echo Done. Artifacts in dist\:
dir /b dist\
goto :eof

:build
set GOOS=%1
set GOARCH=%2
set STAGE=%BINARY%_!VERSION!_%GOOS%_%GOARCH%
set STAGE_DIR=dist\!STAGE!

if not exist "!STAGE_DIR!" mkdir "!STAGE_DIR!"

set EXT=
if "%GOOS%"=="windows" set EXT=.exe

echo ^-^> Building %GOOS%/%GOARCH%...
set CGO_ENABLED=0
set GOOS=%GOOS%
set GOARCH=%GOARCH%
go build -ldflags="!LDFLAGS!" -trimpath -o "!STAGE_DIR!\%BINARY%!EXT!" .
if errorlevel 1 (
    echo    FAILED: %GOOS%/%GOARCH%
    goto :eof
)

copy /y config.example.yaml "!STAGE_DIR!\" >nul

if "%GOOS%"=="windows" (
    powershell -NoProfile -Command "Compress-Archive -Path '!STAGE_DIR!' -DestinationPath 'dist\!STAGE!.zip' -Force"
    echo    dist\!STAGE!.zip
) else (
    tar -czf "dist\!STAGE!.tar.gz" -C dist "!STAGE!"
    echo    dist\!STAGE!.tar.gz
)

rmdir /s /q "!STAGE_DIR!"
goto :eof
