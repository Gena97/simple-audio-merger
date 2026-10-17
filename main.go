package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) == 1 {
		if err := runGUI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	inputDir := flag.String("input", "", "Path to the folder with MP3 and WAV files (including subfolders)")
	outputFile := flag.String("output", "merged_audio.mp3", "Path to the output MP3 file")
	bitrate := flag.String("bitrate", "320", "MP3 bitrate in kbps (e.g., 128, 192, 320)")
	overwrite := flag.Bool("overwrite", false, "Replace an existing output file")
	flag.Parse()

	if *inputDir == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Error: Specify an audio folder using -input")
		flag.Usage()
		os.Exit(1)
	}

	bitrateValue, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(*bitrate)), "k"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Bitrate must be a number in kbps, e.g., 192")
		os.Exit(1)
	}

	paths, err := collectAudioFiles(*inputDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error scanning folder:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err = mergeFiles(ctx, paths, *outputFile, bitrateValue, *overwrite)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error merging audio:", err)
		os.Exit(1)
	}

	fmt.Println("Merging completed! Output file:", *outputFile)
}

func collectAudioFiles(inputDir string) ([]string, error) {
	info, err := os.Stat(inputDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input must be a folder: %s", inputDir)
	}

	var paths []string
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isSupportedAudio(path) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func isSupportedAudio(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3" || ext == ".wav"
}
