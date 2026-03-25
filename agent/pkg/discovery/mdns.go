package discovery

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hashicorp/mdns"
)

// Announce starts the mDNS service to broadcast the Agent's presence on the LAN.
// It returns the running server so it can be gracefully shut down later.
func Announce(port int) (*mdns.Server, error) {
	// Retrieve the machine's hostname for the instance name
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "TermViewer-Agent"
	}

	// Clean the hostname to avoid DNS resolution issues
	instanceName := strings.ReplaceAll(hostname, ".", "-")
	instanceName = strings.ReplaceAll(instanceName, " ", "-")

	// Create the mDNS service
	// Using "" for domain defaults to "local." correctly.
	service, err := mdns.NewMDNSService(
		instanceName,
		"_termviewer._tcp",
		"", 
		"", // Hostname (auto-detected)
		port,
		nil, // IPs (auto-detected)
		[]string{"version=0.1.0"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS service configuration: %w", err)
	}

	// Start the mDNS server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("failed to start mDNS server: %w", err)
	}

	slog.Info("mDNS broadcasting started", "name", instanceName, "port", port)
	return server, nil
}
