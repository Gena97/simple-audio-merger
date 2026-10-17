# Simple Audio Merger

**Simple Audio Merger** combines MP3 and WAV files into a single MP3 file. It includes a native Windows interface and a command-line interface.

## Requirements

- Windows 10 or 11, 64-bit, for the graphical interface
- **FFmpeg**, either available through `PATH` or placed as `ffmpeg.exe` beside the application

FFmpeg is not included in the download. Download it from [ffmpeg.org](https://ffmpeg.org/download.html).

## Installation

1. Download `simple-audio-merger-windows-amd64.zip` from [Releases](https://github.com/Gena97/simple-audio-merger/releases).
2. Extract the archive to a folder.
3. Install FFmpeg or place `ffmpeg.exe` in the same folder as the application.

## Graphical interface

Open `simple-audio-merger.exe` without command-line arguments.

1. Add MP3 or WAV files.
2. Select a file and use the up and down buttons to change its position. Files are merged in the displayed order.
3. Select the output MP3 file and bitrate in kbit/s.
4. Start merging.

You can remove files from the list before merging. The default bitrate is 320 kbit/s.

## Command-line interface

Use `simple-audio-merger-cli.exe` in a terminal. It recursively finds MP3 and WAV files in the input folder and merges them in filename order.

```powershell
.\simple-audio-merger-cli.exe -input "C:\Audio" -output "C:\merged.mp3"
```

Specify bitrate as a number in kbit/s:

```powershell
.\simple-audio-merger-cli.exe -input "C:\Audio" -output "C:\merged.mp3" -bitrate 192
```

Add `-overwrite` to replace an existing output file. Keep the output outside the input folder so it is not picked up as an input on subsequent runs. The application refuses to overwrite an input file.

Display all command-line options:

```powershell
.\simple-audio-merger-cli.exe -help
```

## Build from source

Install Go 1.23.6 or later and run the build script from PowerShell on Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

The script downloads a pinned version of the resource compiler, embeds the Windows manifest, and creates:

- `dist/simple-audio-merger.exe` — graphical application without a console window
- `dist/simple-audio-merger-cli.exe` — console application
- `dist/simple-audio-merger-windows-amd64.zip` — both applications and this README
- `dist/SHA256SUMS` — SHA-256 checksum of the ZIP archive

Internet access is required on the first build to download Go dependencies. FFmpeg is only required when merging files.

## GitHub Actions and releases

The **Build** workflow builds and uploads the Windows x64 package on pushes, pull requests, and manual runs. Download it from the workflow run's **Artifacts** section.

The **Release** workflow builds the same package when a tag beginning with `v` is pushed. It publishes a GitHub release with the ZIP archive and `SHA256SUMS`.

For example, after committing and pushing the changes to be released:

```powershell
git tag v1.0.0
git push origin v1.0.0
```

Choose an unused version tag for each release. The workflows build the application without running tests.
