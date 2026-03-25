package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	relayHeartbeatInterval = 5 * time.Second
	defaultShareTTLSeconds = 300
)

var isCurrentlyStreaming = false

type relayHeartbeatResponse struct {
	Status              string     `json:"status"`
	Refreshed           bool       `json:"refreshed"`
	ShareSessionExpires *time.Time `json:"share_session_expires_at"`
}

// ConnectToRelay establishes an outbound connection to the enterprise relay server.
func ConnectToRelay(relayURL, clientID, clientSecret string, skipTLS bool) error {
	header := http.Header{}
	header.Add("X-Client-ID", clientID)
	header.Add("X-Client-Secret", clientSecret)

	apiBaseURL, err := relayAPIBaseURL(relayURL)
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}
	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

	dialer := *websocket.DefaultDialer
	if skipTLS {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	slog.Info("Connecting to relay server", "url", relayURL, "client_id", clientID, "skip_tls", skipTLS)

	for {
		conn, _, err := dialer.Dial(relayURL, header)
		if err != nil {
			slog.Error("Failed to connect to relay", "error", err)
			slog.Info("Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}

		slog.Info("Successfully connected to relay server")

		ctx, cancel := context.WithCancel(context.Background())
		go startRelayHeartbeat(ctx, httpClient, apiBaseURL, clientID, clientSecret)

		// Reuse the terminal session handler
		isCurrentlyStreaming = true
		HandleTerminalSession(ctx, conn, clientID, "relay-server", true)
		isCurrentlyStreaming = false
		cancel()

		slog.Warn("Relay connection closed. Reconnecting...")
		time.Sleep(2 * time.Second)
	}
}

func startRelayHeartbeat(ctx context.Context, httpClient *http.Client, apiBaseURL, clientID, clientSecret string) {
	sendHeartbeat := func() {
		response, err := postRelayHeartbeat(httpClient, apiBaseURL, clientID, clientSecret, !isCurrentlyStreaming)
		if err != nil {
			slog.Warn("Relay heartbeat failed", "error", err)
			return
		}

		if response.Refreshed {
			slog.Info("Share session refreshed", "status", response.Status, "expires_at", response.ShareSessionExpires)
		}
	}

	sendHeartbeat()

	ticker := time.NewTicker(relayHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHeartbeat()
		}
	}
}

func postRelayHeartbeat(httpClient *http.Client, apiBaseURL, clientID, clientSecret string, available bool) (*relayHeartbeatResponse, error) {
	payload := map[string]interface{}{
		"share_enabled": true,
		"ttl_seconds":   defaultShareTTLSeconds,
		"reset_status":  available,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiBaseURL+"/api/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", clientID)
	req.Header.Set("X-Client-Secret", clientSecret)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("heartbeat failed with status %s", resp.Status)
	}

	var heartbeat relayHeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeat); err != nil {
		return nil, err
	}

	return &heartbeat, nil
}

func relayAPIBaseURL(relayURL string) (string, error) {
	parsed, err := url.Parse(relayURL)
	if err != nil {
		return "", err
	}

	switch parsed.Scheme {
	case "wss":
		parsed.Scheme = "https"
	case "ws":
		parsed.Scheme = "http"
	case "https", "http":
		// already normalized
	default:
		return "", fmt.Errorf("unsupported relay url scheme: %s", parsed.Scheme)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = ""

	return strings.TrimSuffix(parsed.String(), "/"), nil
}
