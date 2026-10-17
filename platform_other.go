//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

func runGUI() error {
	return fmt.Errorf("the graphical interface is available on Windows; use -input to run in command-line mode")
}

func configureCommand(cmd *exec.Cmd) {}
