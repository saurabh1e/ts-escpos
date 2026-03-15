# ...existing code...
### For Windows (32-bit)
```powershell
$Env:GOOS = "windows"
$Env:GOARCH = "386"
go build -o ts-escpos-lite-32.exe cmd/headless/main.go
```

## IMPORTANT: Windows 7 / Embedded Standard 2010 Compatibility
For Windows 7, Windows Embedded Standard 7, or Windows Embedded 2010 compatibility, you **MUST build with Go 1.20 or earlier**.
Go 1.21 and later versions require Windows 10 or Windows Server 2016.

If building on a modern machine with Go 1.21+, the resulting binary will fail to start on older Windows versions (usually with a DLL entry point error).

Recommended build steps for older Windows:
1. Install Go 1.20.14 (the last version to support Windows 7).
2. Clean your build cache: `go clean -cache`
3. Build the lite version as described above.

### For Linux/macOS

### Lite Version (Headless)
The headless version now runs continuously and ignores basic termination signals (like Ctrl+C) to prevent accidental closure.

**Command Line Arguments:**
- `--install-autostart` / `-install`: Adds the application to Windows startup.
- `--remove-autostart` / `-uninstall`: Removes the application from Windows startup.
- `--background` / `-bg`: Hides the console window immediately on startup (Windows only). Use this for a truly invisible background process.

**Stopping the App:**
Since it ignores Ctrl+C, you must terminate the process via Task Manager or the `taskkill` command:
```powershell
taskkill /F /IM ts-escpos-lite.exe
```

### Automation
The release script `scripts/release.sh` now automatically handles this. 
It will detecting that you are building the lite versions and automatically download and use Go 1.20.14 to ensure compatibility, without affecting your main system Go installation.

### Manual Build Steps (if not using release script)
If you need to build manually for older Windows:
1. Install Go 1.20.14 (the last version to support Windows 7).
2. Clean your build cache: `go clean -cache`
3. Set up your environment for 32-bit Windows target:
```powershell
$Env:GOOS = "windows"
$Env:GOARCH = "386"
```
4. Build the lite version:
```bash
go build -o ts-escpos-lite-32.exe cmd/headless/main.go
```
