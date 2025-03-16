# Simple Audio Merger

**Simple Audio Merger** is a simple tool for merging audio files in MP3 and WAV formats. It allows you to combine multiple audio files into one and save the result in MP3 format. The tool also supports custom bitrate settings.

## Requirements

- **FFmpeg** must be installed on your computer for this tool to work.

## Installation

1. Download the `simple-audio-merger.exe`.
2. Ensure that FFmpeg is installed and properly configured on your system.

## Usage

To merge audio files, run the following command:
```
./simple-audio-merger.exe -input "<path_to_input_files>" -output "<path_to_output_file>"
```

You can also use bitrate flag "-bitrate <bitrate_in_kbps>":
```
./simple-audio-merger.exe -input "<path_to_input_files>" -output "<path_to_output_file> -bitrate <bitrate in kbps>"
```
