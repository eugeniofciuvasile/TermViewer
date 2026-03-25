package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Read returns the current content of the system clipboard.
func Read() (string, error) {
	switch runtime.GOOS {
	case "linux":
		// Try wl-paste first (Wayland)
		if _, err := exec.LookPath("wl-paste"); err == nil {
			out, err := exec.Command("wl-paste", "--no-newline").Output()
			if err == nil {
				return string(out), nil
			}
		}
		// Fallback to xclip
		if _, err := exec.LookPath("xclip"); err == nil {
			out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
			if err == nil {
				return string(out), nil
			}
		}
		// Fallback to xsel
		if _, err := exec.LookPath("xsel"); err == nil {
			out, err := exec.Command("xsel", "--clipboard", "--output").Output()
			if err == nil {
				return string(out), nil
			}
		}
	case "darwin":
		out, err := exec.Command("pbpaste").Output()
		if err == nil {
			return string(out), nil
		}
	case "windows":
		// Use PowerShell to get clipboard content
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard").Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("clipboard read not supported or failed on %s", runtime.GOOS)
}

// Write sets the system clipboard content.
func Write(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		// Try wl-copy first
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value $Input")
	}

	if cmd == nil {
		return fmt.Errorf("clipboard write not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}
