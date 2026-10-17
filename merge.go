package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func mergeFiles(ctx context.Context, paths []string, output string, bitrate int, overwrite bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no input audio files selected")
	}
	switch bitrate {
	case 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320:
	default:
		return fmt.Errorf("unsupported MP3 bitrate: %d", bitrate)
	}
	if strings.TrimSpace(output) == "" || !strings.EqualFold(filepath.Ext(output), ".mp3") {
		return fmt.Errorf("output file must have an .mp3 extension")
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	inputs := make([]string, len(paths))
	inputInfo := make([]os.FileInfo, len(paths))
	for i, path := range paths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("input file %d has an empty path", i+1)
		}
		extension := filepath.Ext(path)
		if !strings.EqualFold(extension, ".mp3") && !strings.EqualFold(extension, ".wav") {
			return fmt.Errorf("unsupported input file: %s", path)
		}
		inputs[i], err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve input path %q: %w", path, err)
		}
		inputInfo[i], err = os.Stat(inputs[i])
		if err != nil {
			return fmt.Errorf("read input file %q: %w", path, err)
		}
		if !inputInfo[i].Mode().IsRegular() {
			return fmt.Errorf("input is not a regular file: %s", path)
		}
		if inputs[i] == output {
			return fmt.Errorf("output file must not replace an input file")
		}
	}
	if err := validateMergeOutput(output, inputInfo, overwrite); err != nil {
		return err
	}
	directoryInfo, err := os.Stat(filepath.Dir(output))
	if err != nil {
		return fmt.Errorf("read output folder: %w", err)
	}
	if !directoryInfo.IsDir() {
		return fmt.Errorf("output folder is not a directory")
	}
	ffmpeg, err := findFFmpeg()
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(output), ".simple-audio-merger-*.mp3")
	if err != nil {
		return fmt.Errorf("create temporary output file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output file: %w", err)
	}

	processContext, cancel := context.WithCancel(ctx)
	defer cancel()
	encoder := exec.CommandContext(processContext, ffmpeg,
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "2", "-i", "pipe:0",
		"-map", "0:a:0", "-c:a", "libmp3lame", "-b:a", fmt.Sprintf("%dk", bitrate),
		"-f", "mp3", temporaryPath,
	)
	configureCommand(encoder)
	encoderLog := &limitedDiagnostics{}
	encoder.Stderr = encoderLog
	pcm, err := encoder.StdinPipe()
	if err != nil {
		return fmt.Errorf("create audio pipe: %w", err)
	}
	if err := encoder.Start(); err != nil {
		pcm.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("start MP3 encoder: %w", err)
	}

	// Единый PCM-поток позволяет объединять файлы с разными параметрами
	for _, path := range inputs {
		decoder := exec.CommandContext(processContext, ffmpeg,
			"-nostdin", "-hide_banner", "-loglevel", "error", "-xerror",
			"-i", path, "-map", "0:a:0", "-vn", "-sn", "-dn",
			"-f", "s16le", "-c:a", "pcm_s16le", "-ar", "44100", "-ac", "2", "pipe:1",
		)
		configureCommand(decoder)
		decoderLog := &limitedDiagnostics{}
		decoder.Stderr = decoderLog
		decoder.Stdout = pcm
		if err := decoder.Run(); err != nil {
			pcm.Close()
			cancel()
			encoder.Wait()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			failure := ffmpegFailure(fmt.Sprintf("decode input file %q", path), err, decoderLog)
			if diagnostics := encoderLog.String(); diagnostics != "" {
				return fmt.Errorf("%w; MP3 encoder: %s", failure, diagnostics)
			}
			return failure
		}
	}
	closeErr := pcm.Close()
	encodeErr := encoder.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if encodeErr != nil {
		return ffmpegFailure("encode MP3 output", encodeErr, encoderLog)
	}
	if closeErr != nil {
		return fmt.Errorf("close audio pipe: %w", closeErr)
	}
	if err := validateMergeOutput(output, inputInfo, overwrite); err != nil {
		return err
	}
	if !overwrite {
		// Резервируем свободное имя перед сохранением готового файла
		reserved, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("reserve output file: %w", err)
		}
		reservedInfo, statErr := reserved.Stat()
		closeErr := reserved.Close()
		defer func() {
			current, err := os.Stat(output)
			if err == nil && reservedInfo != nil && os.SameFile(current, reservedInfo) {
				os.Remove(output)
			}
		}()
		if statErr != nil {
			os.Remove(output)
			return fmt.Errorf("read reserved output file: %w", statErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close reserved output file: %w", closeErr)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, output); err != nil {
		return fmt.Errorf("save merged output file: %w", err)
	}
	return nil
}

func validateMergeOutput(output string, inputs []os.FileInfo, overwrite bool) error {
	info, err := os.Stat(output)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read output file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("output is not a regular file: %s", output)
	}
	for _, input := range inputs {
		if os.SameFile(info, input) {
			return fmt.Errorf("output file must not replace an input file")
		}
	}
	if !overwrite {
		return fmt.Errorf("output file already exists: %s", output)
	}
	return nil
}

func findFFmpeg() (string, error) {
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, nil
	}
	executable, err := os.Executable()
	if err == nil {
		name := "ffmpeg"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if path, err := exec.LookPath(filepath.Join(filepath.Dir(executable), name)); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("FFmpeg was not found in PATH or next to the application")
}

type limitedDiagnostics struct {
	data []byte
}

func (diagnostics *limitedDiagnostics) Write(data []byte) (int, error) {
	const limit = 8 * 1024
	length := len(data)
	if length >= limit {
		diagnostics.data = append(diagnostics.data[:0], data[length-limit:]...)
		return length, nil
	}
	if excess := len(diagnostics.data) + length - limit; excess > 0 {
		copy(diagnostics.data, diagnostics.data[excess:])
		diagnostics.data = diagnostics.data[:len(diagnostics.data)-excess]
	}
	diagnostics.data = append(diagnostics.data, data...)
	return length, nil
}

func (diagnostics *limitedDiagnostics) String() string {
	return strings.TrimSpace(string(diagnostics.data))
}

func ffmpegFailure(action string, err error, diagnostics *limitedDiagnostics) error {
	if output := diagnostics.String(); output != "" {
		return fmt.Errorf("%s: %w: %s", action, err, output)
	}
	return fmt.Errorf("%s: %w", action, err)
}
