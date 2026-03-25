package server

import (
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/termviewer/agent/pkg/terminal"
)

// StartUDSServer starts the Unix Domain Socket server for local passwordless attachments.
func StartUDSServer(socketPath string, persistentTerm *terminal.Terminal, command string) error {
	PersistentTerminal = persistentTerm
	defaultCommand = command

	// Clean up old socket if it exists
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/terminal", makeTerminalHandler(true)) // autoAuth = true

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Restrict permissions so only the owner can read/write to the socket
	if err := os.Chmod(socketPath, 0600); err != nil {
		return err
	}

	slog.Info("Starting UDS server", "path", socketPath)
	return http.Serve(listener, mux)
}
