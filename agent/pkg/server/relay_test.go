package server

import "testing"

func TestRelayAPIBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wss relay url",
			input:    "wss://relay.termviewer.example/ws/relay/agent",
			expected: "https://relay.termviewer.example",
		},
		{
			name:     "ws relay url",
			input:    "ws://localhost:3001/ws/relay/agent",
			expected: "http://localhost:3001",
		},
		{
			name:     "http url passthrough",
			input:    "http://localhost:3001",
			expected: "http://localhost:3001",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := relayAPIBaseURL(tc.input)
			if err != nil {
				t.Fatalf("relayAPIBaseURL returned error: %v", err)
			}

			if got != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, got)
			}
		})
	}
}
