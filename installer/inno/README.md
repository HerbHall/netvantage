# SubNetree Scout Windows Installer

Builds a Windows installer for the Scout agent using [Inno Setup 6](https://jrsoftware.org/isinfo.php).

## Prerequisites

- [Inno Setup 6](https://jrsoftware.org/isdl.php) installed on Windows
- Scout binary built for Windows AMD64

## Local Build

1. Build Scout:

   ```bash
   go build -o installer/inno/bin/scout.exe ./cmd/scout/
   ```

2. Set version and compile:

   ```cmd
   set SCOUT_VERSION=1.0.0
   iscc installer\inno\subnetree-scout.iss
   ```

3. Output: `installer/inno/output/SubNetreeScout-1.0.0-setup.exe`

## CI Build

The installer is built automatically by the `build-installer.yml` workflow
when a release is published. It downloads the Scout Windows binary from the
release assets and packages it into the installer.

## What the Installer Does

- Installs `scout.exe` to `C:\Program Files\SubNetree\Scout\`
- Creates Start Menu shortcuts
- Optionally adds the install directory to the system PATH
- Shows the BSL 1.1 license during installation
- Registers in Windows Add/Remove Programs for clean uninstall
- Removes the PATH entry on uninstall

## Installer Details

| Property | Value |
|----------|-------|
| Minimum OS | Windows 10 |
| Architecture | x64 only |
| Privileges | Admin required |
| Compression | LZMA (solid) |
| Output filename | `SubNetreeScout-{version}-setup.exe` |
