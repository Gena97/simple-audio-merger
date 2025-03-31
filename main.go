package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	inputDir := flag.String("input", "", "Path to the folder with audio files")
	outputFile := flag.String("output", "merged_audio.mp3", "Name of the output file")
	bitrate := flag.String("bitrate", "320", "Bitrate for output file (e.g., 128k, 192k, 320k)")
	flag.Parse()

	if *inputDir == "" {
		fmt.Println("Error: You must specify the path to the folder with audio files using -input")
		return
	}

	listFile := "simple-audio-merger-logs.txt"

	// Open file for the list
	f, err := os.Create(listFile)
	if err != nil {
		fmt.Println("Error creating file list:", err)
		return
	}
	defer f.Close()

	fileIndex := 1
	// Search for all audio files in the folder
	err = filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".mp3" || filepath.Ext(path) == ".wav") {
			_, err := f.WriteString(fmt.Sprintf("file '%s'\n", path))
			fileIndex++
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error scanning folder:", err)
		return
	}

	// Run ffmpeg to merge files with specified bitrate
	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile, "-c:a", "libmp3lame", "-b:a", fmt.Sprintf("%sk", *bitrate), *outputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error executing ffmpeg:", err)
		return
	}

	fmt.Println("Merging completed! Output file:", *outputFile)
}
