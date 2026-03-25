package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeShareSessionTTL(t *testing.T) {
	if got := normalizeShareSessionTTL(0); got != defaultShareSessionTTL {
		t.Fatalf("expected default ttl, got %v", got)
	}

	if got := normalizeShareSessionTTL(10); got != minShareSessionTTL {
		t.Fatalf("expected min ttl, got %v", got)
	}

	if got := normalizeShareSessionTTL(int((20 * time.Minute).Seconds())); got != maxShareSessionTTL {
		t.Fatalf("expected max ttl, got %v", got)
	}
}

func TestBuildShareDeepLink(t *testing.T) {
	link := buildShareDeepLink("https://api.termviewer.example", "abc123", "refresh456")

	if !strings.HasPrefix(link, "termviewer://connect?") {
		t.Fatalf("expected termviewer deep link, got %s", link)
	}

	if !strings.Contains(link, "session_token=abc123") {
		t.Fatalf("expected session token in deep link, got %s", link)
	}

	if !strings.Contains(link, "refresh_token=refresh456") {
		t.Fatalf("expected refresh token in deep link, got %s", link)
	}
}
