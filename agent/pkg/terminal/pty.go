package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Terminal represents an active pseudo-terminal session.
type Terminal struct {
	PtyFile *os.File
	cmd     *exec.Cmd

	// Track all connected client sizes for size arbitration
	mu          sync.Mutex
	clientSizes map[string]winsize
	rows        uint16
	cols        uint16

	// Broadcaster logic
	listeners []chan []byte
	OnExit    func()

	// Recording state
	recordingFile  *os.File
	recordingStart time.Time
}

type winsize struct {
	rows uint16
	cols uint16
}

// UpdateClientSize updates the size for a specific client and re-computes the "safe" PTY size.
func (t *Terminal) UpdateClientSize(clientID string, rows, cols uint16) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.clientSizes == nil {
		t.clientSizes = make(map[string]winsize)
	}

	if rows == 0 || cols == 0 {
		delete(t.clientSizes, clientID)
	} else {
		t.clientSizes[clientID] = winsize{rows: rows, cols: cols}
	}

	// Compute the "Smallest Common Denominator" to ensure all clients can see the full output
	// without wrapping or broken TUI redraws.
	var minRows, minCols uint16 = 0xFFFF, 0xFFFF
	if len(t.clientSizes) == 0 {
		return nil
	}

	for _, sz := range t.clientSizes {
		if sz.rows < minRows {
			minRows = sz.rows
		}
		if sz.cols < minCols {
			minCols = sz.cols
		}
	}

	// Only resize if the computed "safe" size has actually changed
	if t.rows == minRows && t.cols == minCols {
		return nil
	}

	t.rows = minRows
	t.cols = minCols
	return pty.Setsize(t.PtyFile, &pty.Winsize{
		Rows: minRows,
		Cols: minCols,
	})
}

// StartShell spawns a command (or the default OS shell) and attaches it to a PTY.
func StartShell(rows, cols uint16, command string) (*Terminal, error) {
	var cmd *exec.Cmd
	if command != "" {
		// Use /bin/sh -c to allow complex commands like "tmux attach -t 1"
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
	} else {
		shell := "bash"
		if runtime.GOOS == "windows" {
			shell = "powershell.exe"
		} else if userShell := os.Getenv("SHELL"); userShell != "" {
			shell = userShell
		}
		cmd = exec.Command(shell)
	}

	// Set standard environment for better terminal compatibility
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	if rows > 0 && cols > 0 {
		pty.Setsize(ptyFile, &pty.Winsize{
			Rows: rows,
			Cols: cols,
		})
	}

	t := &Terminal{
		PtyFile: ptyFile,
		cmd:     cmd,
		rows:    rows,
		cols:    cols,
	}

	// Start the background broadcast loop
	go t.broadcastLoop()

	return t, nil
}

// broadcastLoop reads from the PTY and sends data to all active listeners.
func (t *Terminal) broadcastLoop() {
	buf := make([]byte, 1024)
	for {
		n, err := t.PtyFile.Read(buf)
		if err != nil {
			break
		}
		
		data := make([]byte, n)
		copy(data, buf[:n])

		t.mu.Lock()
		// Record if active
		if t.recordingFile != nil {
			timestamp := time.Since(t.recordingStart).Seconds()
			// JSON escape data for asciinema format
			line := fmt.Sprintf("[%f, \"o\", %q]\n", timestamp, string(data))
			t.recordingFile.WriteString(line)
		}

		for _, listener := range t.listeners {
			select {
			case listener <- data:
			default:
			}
		}
		t.mu.Unlock()
	}

	// Clean up: Close all listener channels to signal they are done
	t.mu.Lock()
	for _, listener := range t.listeners {
		close(listener)
	}
	t.listeners = nil
	t.mu.Unlock()

	if t.OnExit != nil {
		t.OnExit()
	}
}

// AddListener creates and returns a new data channel for this terminal.
func (t *Terminal) AddListener() chan []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	ch := make(chan []byte, 64)
	t.listeners = append(t.listeners, ch)
	return ch
}

// RemoveListener closes and removes a data channel.
func (t *Terminal) RemoveListener(ch chan []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, l := range t.listeners {
		if l == ch {
			t.listeners = append(t.listeners[:i], t.listeners[i+1:]...)
			break
		}
	}
}


func (t *Terminal) Close() error {
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	if t.PtyFile != nil {
		return t.PtyFile.Close()
	}
	return nil
}

// Write implements io.Writer to pipe data into the PTY stdin.
func (t *Terminal) Write(p []byte) (n int, err error) {
	return t.PtyFile.Write(p)
}

// StartRecording starts recording the terminal session to a file in Asciinema format.
func (t *Terminal) StartRecording(filePath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.recordingFile != nil {
		return fmt.Errorf("recording already in progress")
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	// Write Asciinema v2 header
	header := fmt.Sprintf("{\"version\": 2, \"width\": %d, \"height\": %d, \"timestamp\": %d, \"env\": {\"TERM\": \"xterm-256color\"}}\n", t.cols, t.rows, time.Now().Unix())
	f.WriteString(header)

	t.recordingFile = f
	t.recordingStart = time.Now()
	return nil
}

// StopRecording stops the current recording and closes the file.
func (t *Terminal) StopRecording() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.recordingFile != nil {
		t.recordingFile.Sync()
		t.recordingFile.Close()
		t.recordingFile = nil
	}
}

// IsRecording returns true if the session is currently being recorded.
func (t *Terminal) IsRecording() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.recordingFile != nil
}
