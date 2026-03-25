package terminal

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Session represents a discoverable terminal session (tmux, etc.)
type Session struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // "tmux", "process", etc.
	Context    string `json:"context"`
	IsAttached bool   `json:"is_attached"`
}

// DiscoverSessions scans the system for active terminal sessions.
func DiscoverSessions() ([]Session, error) {
	sessions := []Session{}

	// 0. Discover native termviewer sessions
	sessions = append(sessions, ListNativeSessions()...)

	// 1. Discover tmux sessions
	tmuxSessions, err := listTmuxSessions()
	if err == nil {
		sessions = append(sessions, tmuxSessions...)
	}

	return sessions, nil
}

func listProcessSessions() ([]Session, error) {
	windowMap := make(map[string]string)
	wmOut, err := exec.Command("wmctrl", "-l", "-p").Output()
	if err == nil {
		lines := strings.Split(string(wmOut), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				pid := fields[2]
				title := strings.Join(fields[3:], " ")
				windowMap[pid] = title
			}
		}
	}

	cmd := exec.Command("ps", "-eo", "pid,stat,tty,comm", "--no-headers")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	sessions := []Session{}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		pid := fields[0]
		stat := fields[1]
		tty := fields[2]
		comm := fields[3]

		if tty == "?" || tty == "-" || !strings.ContainsAny(stat, "s+") {
			continue
		}

		if comm == "bash" || comm == "zsh" || comm == "fish" {
			name := ""
			currPid := pid
			for i := 0; i < 5; i++ {
				if title, ok := windowMap[currPid]; ok {
					name = title
					break
				}
				ppidCmd := exec.Command("ps", "-o", "ppid=", "-p", currPid)
				if ppidOut, err := ppidCmd.Output(); err == nil {
					currPid = strings.TrimSpace(string(ppidOut))
				} else {
					break
				}
			}

			if name == "" {
				name = fmt.Sprintf("%s (PID %s on %s)", comm, pid, tty)
			}

			cwd := ""
			cwdLink, err := exec.Command("readlink", fmt.Sprintf("/proc/%s/cwd", pid)).Output()
			if err == nil {
				cwd = strings.TrimSpace(string(cwdLink))
			} else {
				cwd = fmt.Sprintf("Running %s on %s", comm, tty)
			}

			sessions = append(sessions, Session{
				ID:         "proc:" + pid,
				Name:       name,
				Type:       "terminal",
				Context:    cwd,
				IsAttached: true,
			})
		}
	}

	return sessions, nil
}

func listTmuxSessions() ([]Session, error) {
	// Format: session_name|attached|created|last_attached
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}|#{session_attached}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	sessions := []Session{}

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		attached := parts[1] == "1"

		// Capture a bit of context
		context, _ := captureTmuxContext(name)

		sessions = append(sessions, Session{
			ID:         "tmux:" + name,
			Name:       "tmux: " + name,
			Type:       "tmux",
			Context:    context,
			IsAttached: attached,
		})
	}

	return sessions, nil
}

func captureTmuxContext(name string) (string, error) {
	// Capture the last 20 lines
	cmd := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", "-20")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
