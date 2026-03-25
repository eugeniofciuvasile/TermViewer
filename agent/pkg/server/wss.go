package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/termviewer/agent/pkg/clipboard"
	"github.com/termviewer/agent/pkg/security"
	"github.com/termviewer/agent/pkg/terminal"
)

// Global configuration for the server
var appPasswordHash string
var defaultCommand string
var PersistentTerminal *terminal.Terminal
var AuthRateLimiter = security.NewRateLimiter(5, 5*time.Minute)

// upgrader configures how HTTP requests are upgraded to WebSockets.
var upgrader = websocket.Upgrader{
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		// Allow any origin for local LAN connections.
		return true
	},
}

// StartWSServer starts the WebSocket Secure (WSS) HTTP server.
func StartWSServer(port int, certFile, keyFile, passwordHash string, persistentTerm *terminal.Terminal, command string) error {
	appPasswordHash = passwordHash
	PersistentTerminal = persistentTerm
	defaultCommand = command
	mux := http.NewServeMux()
	mux.HandleFunc("/terminal", makeTerminalHandler(false))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}

	slog.Info("Starting WSS server", "url", fmt.Sprintf("wss://0.0.0.0:%d/terminal", port))
	return srv.ListenAndServeTLS(certFile, keyFile)
}

func makeTerminalHandler(autoAuth bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		clientID := r.RemoteAddr
		if autoAuth {
			clientIP = "127.0.0.1"
			clientID = "local-uds-client"
		}

		if AuthRateLimiter.IsBlocked(clientIP) {
			slog.Warn("Connection rejected: IP is blocked due to too many failed auth attempts", "ip", clientIP)
			http.Error(w, "Too many failed attempts. Try again later.", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Failed to upgrade connection to WebSocket", "error", err)
			return
		}
		defer conn.Close()

		HandleTerminalSession(r.Context(), conn, clientID, clientIP, autoAuth)
	}
}

// HandleTerminalSession manages the terminal session for a given websocket connection.
func HandleTerminalSession(ctx context.Context, conn *websocket.Conn, clientID, clientIP string, autoAuth bool) {
	// Enable compression for this connection
	conn.SetCompressionLevel(1) // Level 1 (best speed) is usually best for terminal data

	// Handle terminal session for a given websocket connection.
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	slog.Info("New terminal session started", "client_id", clientID, "auto_auth", autoAuth)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var term *terminal.Terminal
	isAuthenticated := autoAuth
	var currentNonce string
	var isPersistent bool
	var currentAttachmentID int
	var lastSyncedClipboard string

	// Create a buffered channel for outbound messages
	msgChan := make(chan struct {
		mType int
		data  []byte
	}, 2000)
	msgChanClosed := false
	var msgChanMu sync.Mutex
	var writeMu sync.Mutex // Mutex for the underlying WebSocket connection

	defer func() {
		msgChanMu.Lock()
		msgChanClosed = true
		close(msgChan)
		msgChanMu.Unlock()
	}()

	// Start a dedicated write loop
	go func() {
		for msg := range msgChan {
			writeMu.Lock()
			conn.SetWriteDeadline(time.Now().Add(20 * time.Second))
			err := conn.WriteMessage(msg.mType, msg.data)
			writeMu.Unlock()
			if err != nil {
				slog.Error("Write loop failed", "client_id", clientID, "error", err)
				cancel()
				return
			}
		}
	}()

	// Helper for non-blocking outbound messages
	queueMessage := func(mType int, data []byte) {
		if mType == websocket.PingMessage || mType == websocket.PongMessage || mType == websocket.CloseMessage {
			writeMu.Lock()
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			conn.WriteMessage(mType, data)
			writeMu.Unlock()
			return
		}

		msgChanMu.Lock()
		if msgChanClosed {
			msgChanMu.Unlock()
			return
		}

		select {
		case msgChan <- struct {
			mType int
			data  []byte
		}{mType, data}:
			msgChanMu.Unlock()
		default:
			msgChanMu.Unlock()
			
			if mType == websocket.BinaryMessage {
				// Block until space is available to propagate backpressure to the PTY.
				msgChan <- struct {
					mType int
					data  []byte
				}{mType, data}
				return
			}
			
			slog.Warn("Outbound buffer full, dropping control message", "client_id", clientID, "type", mType)
		}
	}

	// Helper for synchronized JSON writes
	writeJSON := func(v interface{}) error {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		queueMessage(websocket.TextMessage, data)
		return nil
	}

	// Helper for synchronized message writes
	writeMessage := func(msgType int, data []byte) error {
		queueMessage(msgType, data)
		return nil
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				queueMessage(websocket.PingMessage, nil)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !isAuthenticated {
					continue
				}
				text, err := clipboard.Read()
				if err == nil && text != "" && text != lastSyncedClipboard {
					lastSyncedClipboard = text
					writeJSON(map[string]interface{}{
						"type": "clipboard_sync",
						"payload": map[string]interface{}{
							"text": text,
						},
					})
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !isAuthenticated {
					continue
				}
				stats := getSystemStats()
				writeJSON(map[string]interface{}{
					"type":    "system_stats_sync",
					"payload": stats,
				})
			}
		}
	}()

	defer func() {
		if term != nil {
			term.UpdateClientSize(clientID, 0, 0) // Remove this client's size constraints
			if !isPersistent && PersistentTerminal == nil {
				slog.Info("Closing ephemeral PTY session", "client_id", clientID)
				term.Close()
			} else {
				slog.Info("Client detached from persistent session", "client_id", clientID)
			}
		}
	}()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			slog.Info("Client disconnected", "client_id", clientID, "error", err)
			break
		}

		// Handle raw user input (keystrokes) sent as binary data
		if messageType == websocket.BinaryMessage {
			if isAuthenticated && term != nil && term.PtyFile != nil {
				term.PtyFile.Write(message)
			}
			continue
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				slog.Warn("Received invalid JSON", "client_id", clientID, "error", err)
				continue
			}

			msgType, _ := msg["type"].(string)
			payload, _ := msg["payload"].(map[string]interface{})

			switch msgType {
			case "auth_request":
				nonceBytes := make([]byte, 16)
				rand.Read(nonceBytes)
				currentNonce = base64.StdEncoding.EncodeToString(nonceBytes)

				writeJSON(map[string]interface{}{
					"type": "auth_challenge",
					"payload": map[string]interface{}{
						"nonce":     currentNonce,
						"timestamp": time.Now().Format(time.RFC3339),
					},
				})

			case "auth_response":
				if currentNonce == "" {
					slog.Warn("Received auth_response without a preceding challenge", "client_id", clientID)
					continue
				}

				receivedResponse, _ := payload["response"].(string)

				// Verify HMAC-SHA256
				h := hmac.New(sha256.New, []byte(appPasswordHash))
				h.Write([]byte(currentNonce))
				expectedResponse := base64.StdEncoding.EncodeToString(h.Sum(nil))

				if receivedResponse == expectedResponse {
					isAuthenticated = true
					AuthRateLimiter.Reset(clientIP) // Reset on success
					slog.Info("Authentication successful", "client_id", clientID)
					writeJSON(map[string]interface{}{
						"type": "auth_status",
						"payload": map[string]interface{}{
							"status":        "success",
							"session_token": "mock_jwt_session_token",
						},
					})
				} else {
					AuthRateLimiter.RecordFail(clientIP)
					slog.Warn("Authentication FAILED", "client_id", clientID, "ip", clientIP)
					writeJSON(map[string]interface{}{
						"type": "auth_status",
						"payload": map[string]interface{}{
							"status": "failed",
							"reason": "Invalid password",
						},
					})
					return // Close connection on auth failure
				}

			case "session_list_request":
				if !isAuthenticated {
					slog.Warn("Client attempted session_list_request without authentication", "client_id", clientID)
					return
				}
				sessions, err := terminal.DiscoverSessions()
				if err != nil {
					slog.Error("Failed to discover sessions", "client_id", clientID, "error", err)
				}
				writeJSON(map[string]interface{}{
					"type": "session_list_response",
					"payload": map[string]interface{}{
						"sessions": sessions,
					},
				})

			case "terminal_init":
				if !isAuthenticated {
					slog.Warn("Client attempted terminal_init without authentication", "client_id", clientID)
					return
				}

				if term != nil {
					slog.Info("Client switching sessions. Detaching from current.", "client_id", clientID)
					term.UpdateClientSize(clientID, 0, 0)
					term = nil
					isPersistent = false
				}

				rows, cols := uint16(24), uint16(80)
				cmd := defaultCommand
				var sessionID string
				if payload != nil {
					if r, ok := payload["rows"].(float64); ok {
						rows = uint16(r)
					}
					if c, ok := payload["cols"].(float64); ok {
						cols = uint16(c)
					}
					if customCmd, ok := payload["command"].(string); ok && customCmd != "" {
						cmd = customCmd
					}
					if sid, ok := payload["session_id"].(string); ok {
						sessionID = sid
					}
				}

				if PersistentTerminal != nil {
					term = PersistentTerminal
					slog.Info("Client attached to persistent PTY session", "client_id", clientID)
				} else {
					if sessionID != "" {
						if strings.HasPrefix(sessionID, "termviewer:") {
							name := strings.TrimPrefix(sessionID, "termviewer:")
							t, err := terminal.GetOrCreateNativeSession(name, rows, cols, cmd)
							if err != nil {
								slog.Error("Failed to spawn native shell", "client_id", clientID, "session_id", sessionID, "error", err)
								writeJSON(map[string]interface{}{
									"type":    "error",
									"payload": map[string]string{"message": "Shell spawn failed"},
								})
								continue
							}
							term = t
							isPersistent = true
							slog.Info("Attached to native TermViewer session", "client_id", clientID, "session_name", name)
						} else if strings.HasPrefix(sessionID, "tmux:") {
							name := strings.TrimPrefix(sessionID, "tmux:")
							cmd = fmt.Sprintf("tmux attach -t %s", name)
							isPersistent = true
						}
					}

					if term == nil {
						t, err := terminal.StartShell(rows, cols, cmd)
						if err != nil {
							slog.Error("Failed to spawn shell", "client_id", clientID, "error", err)
							writeJSON(map[string]interface{}{
								"type":    "error",
								"payload": map[string]string{"message": "Shell spawn failed"},
							})
							continue
						}
						term = t
						slog.Info("Shell successfully spawned (attached to PTY)", "client_id", clientID)
					}
				}

				currentAttachmentID++
				attachmentID := currentAttachmentID

				go func(targetTerm *terminal.Terminal, id int) {
					writeJSON(map[string]interface{}{
						"type": "terminal_status_response",
						"payload": map[string]interface{}{
							"is_recording": targetTerm.IsRecording(),
						},
					})

					ch := targetTerm.AddListener()
					defer targetTerm.RemoveListener(ch)

					targetTerm.UpdateClientSize(clientID, rows, cols)

					var buffer []byte
					flushTicker := time.NewTicker(5 * time.Millisecond)
					defer flushTicker.Stop()

					for {
						select {
						case data, ok := <-ch:
							if !ok {
								goto EXIT_LOOP
							}
							if term != targetTerm || currentAttachmentID != id {
								return
							}
							buffer = append(buffer, data...)
							if len(buffer) > 4096 {
								writeMessage(websocket.BinaryMessage, buffer)
								buffer = nil
							}
						case <-flushTicker.C:
							if len(buffer) > 0 {
								writeMessage(websocket.BinaryMessage, buffer)
								buffer = nil
							}
						}
					}

				EXIT_LOOP:
					if term == targetTerm && currentAttachmentID == id {
						slog.Info("Terminal session exited. Notifying client.", "client_id", clientID)
						writeJSON(map[string]interface{}{
							"type": "session_closed",
							"payload": map[string]interface{}{
								"reason": "Shell exited",
							},
						})
					}
				}(term, attachmentID)

			case "terminal_resize":
				if isAuthenticated && term != nil && payload != nil {
					rows, rOk := payload["rows"].(float64)
					cols, cOk := payload["cols"].(float64)
					if rOk && cOk {
						term.UpdateClientSize(clientID, uint16(rows), uint16(cols))
					}
				}

			case "clipboard_update":
				if isAuthenticated {
					text, _ := payload["text"].(string)
					if text != "" && text != lastSyncedClipboard {
						lastSyncedClipboard = text
						clipboard.Write(text)
						slog.Info("Clipboard synced from client", "client_id", clientID)
					}
				}

			case "file_list_request":
				if isAuthenticated {
					path, _ := payload["path"].(string)
					absPath, files, err := listFiles(path)
					if err != nil {
						writeJSON(map[string]interface{}{
							"type":    "error",
							"payload": map[string]string{"message": "Failed to list files: " + err.Error()},
						})
					} else {
						writeJSON(map[string]interface{}{
							"type": "file_list_response",
							"payload": map[string]interface{}{
								"path":  absPath,
								"files": files,
							},
						})
					}
				}

			case "file_download_request":
				if isAuthenticated {
					path, _ := payload["path"].(string)
					go streamFileToClient(path, clientID, writeJSON)
				}

			case "file_upload_start":
				if isAuthenticated {
					tid, _ := payload["transfer_id"].(string)
					fname, _ := payload["filename"].(string)
					dpath, _ := payload["destination_path"].(string)
					err := handleFileUploadStart(tid, fname, dpath)
					if err != nil {
						slog.Error("Failed to start file upload", "error", err)
						writeJSON(map[string]interface{}{
							"type":    "error",
							"payload": map[string]string{"message": "Upload start failed: " + err.Error()},
						})
					}
				}

			case "file_data":
				if isAuthenticated {
					tid, _ := payload["transfer_id"].(string)
					data, _ := payload["data"].(string)
					isLast, _ := payload["is_last"].(bool)
					err := handleIncomingFileData(tid, data, isLast)
					if err != nil {
						slog.Error("Error handling incoming file data", "error", err)
					}
				}

			case "terminal_record_toggle":
				if isAuthenticated && term != nil {
					active, _ := payload["active"].(bool)
					if active {
						timestamp := time.Now().Format("20060102-150405")
						filename := fmt.Sprintf("termviewer-record-%s.cast", timestamp)
						err := term.StartRecording(filename)
						if err != nil {
							slog.Error("Failed to start recording", "error", err)
						} else {
							slog.Info("Recording started", "file", filename)
						}
					} else {
						term.StopRecording()
						slog.Info("Recording stopped")
					}

					// Notify client of new status
					writeJSON(map[string]interface{}{
						"type": "terminal_status_response",
						"payload": map[string]interface{}{
							"is_recording": term.IsRecording(),
						},
					})
				}

			case "ping":
				writeJSON(map[string]interface{}{"type": "pong"})

			case "pong":
				// Application-level pong received, nothing to do

			default:
				slog.Warn("Received unhandled control message type", "client_id", clientID, "type", msgType)
			}
		}
	}
}
