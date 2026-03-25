package terminal

import (
	"sync"
)

var (
	mu       sync.Mutex
	sessions = make(map[string]*Terminal)
)

// GetOrCreateNativeSession returns an existing native TermViewer session or creates a new one.
func GetOrCreateNativeSession(name string, rows, cols uint16, cmd string) (*Terminal, error) {
	mu.Lock()
	defer mu.Unlock()

	if t, exists := sessions[name]; exists {
		return t, nil
	}

	t, err := StartShell(rows, cols, cmd)
	if err != nil {
		return nil, err
	}

	t.OnExit = func() {
		mu.Lock()
		if sessions[name] == t {
			delete(sessions, name)
		}
		mu.Unlock()
	}

	sessions[name] = t
	return t, nil
}

// ListNativeSessions returns a list of all active native TermViewer sessions.
func ListNativeSessions() []Session {
	mu.Lock()
	defer mu.Unlock()

	var list []Session
	for name := range sessions {
		list = append(list, Session{
			ID:         "termviewer:" + name,
			Name:       "termviewer: " + name,
			Type:       "termviewer",
			Context:    "Native TermViewer Multiplexer Session",
			IsAttached: true,
		})
	}
	return list
}
