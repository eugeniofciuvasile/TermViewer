package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/termviewer/agent/pkg/config"
	"github.com/termviewer/agent/pkg/discovery"
	"github.com/termviewer/agent/pkg/server"
	ptytls "github.com/termviewer/agent/pkg/tls" // aliased to avoid collision
	"golang.org/x/term"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	passwordFlag := flag.String("password", os.Getenv("TERMVIEWER_PASSWORD"), "Optional override password for mobile app authentication")
	portFlag := flag.Int("port", 24242, "Port for the WebSocket server")
	commandFlag := flag.String("command", "", "Custom command to run (e.g., 'tmux attach -t 1')")
	attachFlag := flag.String("attach", "", "Attach to a native TermViewer session (acts as a local client)")

	relayURL := flag.String("relay-url", os.Getenv("TERMVIEWER_RELAY_URL"), "Enterprise Relay Server WSS URL")
	clientID := flag.String("client-id", os.Getenv("TERMVIEWER_CLIENT_ID"), "Enterprise Client ID")
	clientSecret := flag.String("client-secret", os.Getenv("TERMVIEWER_CLIENT_SECRET"), "Enterprise Client Secret")
	skipTLS := flag.Bool("tls-skip-verify", os.Getenv("TERMVIEWER_TLS_SKIP_VERIFY") == "true", "Skip TLS verification for enterprise relay")

	flag.Parse()

	// Handle local attach
	if *attachFlag != "" {
		runLocalClient(*attachFlag)
		return
	}

	slog.Info("Starting TermViewer Agent Daemon...")

	cfg, err := config.GetOrCreateConfig(*passwordFlag)
	if err != nil {
		slog.Error("Failed to initialize configuration", "error", err)
		os.Exit(1)
	}

	// Merge CLI flags into config – only flags explicitly passed override saved values.
	dirty := false
	explicitFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { explicitFlags[f.Name] = true })

	if explicitFlags["relay-url"] && *relayURL != cfg.RelayURL {
		cfg.RelayURL = *relayURL
		dirty = true
	}
	if explicitFlags["client-id"] && *clientID != cfg.ClientID {
		cfg.ClientID = *clientID
		dirty = true
	}
	if explicitFlags["client-secret"] && *clientSecret != cfg.ClientSecret {
		cfg.ClientSecret = *clientSecret
		dirty = true
	}
	if explicitFlags["tls-skip-verify"] && *skipTLS != cfg.TLSSkipVerify {
		cfg.TLSSkipVerify = *skipTLS
		dirty = true
	}
	if explicitFlags["port"] && *portFlag != cfg.Port {
		cfg.Port = *portFlag
		dirty = true
	}
	if explicitFlags["command"] && *commandFlag != cfg.Command {
		cfg.Command = *commandFlag
		dirty = true
	}

	if dirty {
		if err := cfg.Save(); err != nil {
			slog.Warn("Failed to persist updated configuration", "error", err)
		} else {
			slog.Info("Configuration updated and saved")
		}
	}

	// Use saved values as fallback for flags not provided on CLI
	relayAddr := cfg.RelayURL
	machineID := cfg.ClientID
	machineSecret := cfg.ClientSecret
	tlsSkip := cfg.TLSSkipVerify
	port := *portFlag
	if port == 24242 && cfg.Port != 0 {
		port = cfg.Port
	}
	command := *commandFlag
	if command == "" {
		command = cfg.Command
	}

	certPath := "cert.pem"
	keyPath := "key.pem"

	if _, err := os.Stat(certPath); err == nil {
		slog.Info("Found existing TLS certificate.")
	} else {
		slog.Info("TLS certificate not found. Generating a new self-signed ECDSA P-256 certificate...")
		if err := ptytls.GenerateSelfSignedCert(certPath, keyPath); err != nil {
			slog.Error("Failed to generate certificate", "error", err)
			os.Exit(1)
		}
		slog.Info("TLS certificate and key generated successfully.")
	}

	mdnsServer, err := discovery.Announce(port)
	if err != nil {
		slog.Error("Failed to start mDNS broadcast", "error", err)
		os.Exit(1)
	}
	defer mdnsServer.Shutdown()

	go func() {
		if err := server.StartWSServer(port, certPath, keyPath, cfg.PasswordHash, nil, command); err != nil {
			slog.Error("WSS Server failed", "error", err)
			os.Exit(1)
		}
	}()

	udsPath := filepath.Join(os.TempDir(), "termviewer.sock")
	go func() {
		if err := server.StartUDSServer(udsPath, nil, command); err != nil {
			slog.Error("UDS Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Start enterprise relay if configured
	if relayAddr != "" && machineID != "" && machineSecret != "" {
		go func() {
			if err := server.ConnectToRelay(relayAddr, machineID, machineSecret, tlsSkip); err != nil {
				slog.Error("Relay connection failed", "error", err)
			}
		}()
	}

	// Log fingerprint for TOFU
	logCertificateFingerprint(certPath, keyPath)

	slog.Info("TermViewer Agent Daemon is running", "pid", os.Getpid(), "port", port)
	slog.Info("Discoverable via mDNS. WSS listening.")
	slog.Info("Use the mobile app or './agent --attach main' to connect locally.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutting down TermViewer Agent...")
}

// logCertificateFingerprint logs the SHA-256 fingerprint of the TLS certificate for user verification.
func logCertificateFingerprint(certPath, keyPath string) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		slog.Warn("Failed to load certificate for fingerprinting", "error", err)
		return
	}
	if len(cert.Certificate) == 0 {
		return
	}
	hash := sha256.Sum256(cert.Certificate[0])
	fingerprint := hex.EncodeToString(hash[:])

	slog.Info("SECURITY: TLS CERTIFICATE FINGERPRINT (SHA-256)", "fingerprint", fingerprint)
}

func runLocalClient(sessionName string) {
	socketPath := filepath.Join(os.TempDir(), "termviewer.sock")
	u := url.URL{Scheme: "ws", Host: "localhost", Path: "/terminal"}
	slog.Info("Connecting to local daemon via UDS", "path", socketPath)

	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		slog.Error("dial failed (is the agent running?)", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	width, height := 80, 24
	if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		width, height = w, h
	}
	c.WriteJSON(map[string]interface{}{
		"type": "terminal_init",
		"payload": map[string]interface{}{
			"rows":       height,
			"cols":       width,
			"session_id": "termviewer:" + sessionName,
		},
	})

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		slog.Error("Failed to set terminal to raw mode", "error", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	sigWinch := make(chan os.Signal, 1)
	signal.Notify(sigWinch, syscall.SIGWINCH)
	go func() {
		for range sigWinch {
			if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
				c.WriteJSON(map[string]interface{}{
					"type": "terminal_resize",
					"payload": map[string]interface{}{
						"rows": h,
						"cols": w,
					},
				})
			}
		}
	}()

	done := make(chan struct{})

	// Read from WebSocket, write to Stdout
	go func() {
		defer close(done)
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.BinaryMessage {
				os.Stdout.Write(message)
			} else if mt == websocket.TextMessage {
				var msg map[string]interface{}
				if err := json.Unmarshal(message, &msg); err == nil {
					if msg["type"] == "session_closed" {
						return
					}
				}
			}
		}
	}()

	// Read from Stdin, write to WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			c.WriteMessage(websocket.BinaryMessage, buf[:n])
		}
	}()

	<-done
}
