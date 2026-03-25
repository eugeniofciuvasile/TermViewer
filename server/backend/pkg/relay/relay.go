package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/gofiber/contrib/websocket"
	"gorm.io/gorm"
)

// SafeConn wraps a websocket connection with a mutex and a buffered message channel
type SafeConn struct {
	Conn    *websocket.Conn
	MsgChan chan WebSocketMsg
	closed  bool
	mutex   sync.Mutex
}

type WebSocketMsg struct {
	Type int
	Data []byte
}

func (sc *SafeConn) WriteMessage(messageType int, data []byte) error {
	if messageType == websocket.PingMessage || messageType == websocket.PongMessage || messageType == websocket.CloseMessage {
		sc.mutex.Lock()
		defer sc.mutex.Unlock()
		if sc.closed {
			return websocket.ErrCloseSent
		}
		sc.Conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		return sc.Conn.WriteMessage(messageType, data)
	}

	sc.mutex.Lock()
	isClosed := sc.closed
	sc.mutex.Unlock()

	if isClosed {
		return websocket.ErrCloseSent
	}

	select {
	case sc.MsgChan <- WebSocketMsg{Type: messageType, Data: data}:
		return nil
	default:
		if messageType == websocket.BinaryMessage {
			// Apply backpressure to prevent ANSI corruption
			sc.MsgChan <- WebSocketMsg{Type: messageType, Data: data}
			return nil
		}
		return fmt.Errorf("write buffer full (dropping control message)")
	}
}

func (sc *SafeConn) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return sc.WriteMessage(websocket.TextMessage, data)
}

func (sc *SafeConn) Close() error {
	sc.mutex.Lock()
	if sc.closed {
		sc.mutex.Unlock()
		return nil
	}
	sc.closed = true
	close(sc.MsgChan)
	sc.mutex.Unlock()
	return sc.Conn.Close()
}

func (sc *SafeConn) StartWriteLoop() {
	for msg := range sc.MsgChan {
		sc.mutex.Lock()
		sc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := sc.Conn.WriteMessage(msg.Type, msg.Data)
		sc.mutex.Unlock()
		if err != nil {
			log.Printf("Write loop error: %v", err)
			return
		}
	}
}
func (sc *SafeConn) ReadMessage() (messageType int, p []byte, err error) {
	return sc.Conn.ReadMessage()
}

type RelaySession struct {
	AgentConn *SafeConn
	AppConns  map[string]*SafeConn
	Active    bool
	Mutex     sync.Mutex
}

type RelayEngine struct {
	Sessions map[string]*RelaySession // ClientID -> Session
	Mutex    sync.Mutex
}

func InitRelayEngine() *RelayEngine {
	return &RelayEngine{
		Sessions: make(map[string]*RelaySession),
	}
}

func (e *RelayEngine) RegisterAgent(clientID string, conn *websocket.Conn) {
	e.Mutex.Lock()
	session, ok := e.Sessions[clientID]
	if !ok {
		session = &RelaySession{
			AppConns: make(map[string]*SafeConn),
		}
		e.Sessions[clientID] = session
	}
	e.Mutex.Unlock()

	safeConn := &SafeConn{
		Conn:    conn,
		MsgChan: make(chan WebSocketMsg, 2000), // Large buffer for terminal data
	}

	session.Mutex.Lock()
	if session.AgentConn != nil {
		session.AgentConn.Close()
	}
	session.AgentConn = safeConn
	session.Mutex.Unlock()

	log.Printf("Agent %s registered with relay", clientID)

	// Keep-Alive: Detect dead agents fast (30s timeout)
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	// Start dedicated write loop
	go safeConn.StartWriteLoop()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			session.Mutex.Lock()
			if session.AgentConn != safeConn {
				session.Mutex.Unlock()
				return
			}
			// Use native ping - this bypasses MsgChan for high priority
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				session.Mutex.Unlock()
				log.Printf("Relay: Ping failed for agent %s, cleaning up", clientID)
				e.cleanupAgent(clientID, safeConn)
				return
			}
			session.Mutex.Unlock()
		}
	}()

	// Listen for messages from Agent (Blocks)
	e.proxyAgent(clientID, safeConn)
}

func (e *RelayEngine) ConnectApp(clientID string, conn *websocket.Conn) {
	e.Mutex.Lock()
	session, ok := e.Sessions[clientID]
	e.Mutex.Unlock()

	safeConn := &SafeConn{
		Conn:    conn,
		MsgChan: make(chan WebSocketMsg, 1000),
	}

	if !ok {
		log.Printf("App tried to connect to offline Agent %s", clientID)
		conn.WriteJSON(map[string]string{"error": "Agent offline"})
		conn.Close()
		return
	}

	appID := conn.RemoteAddr().String()

	session.Mutex.Lock()
	if session.AppConns == nil {
		session.AppConns = make(map[string]*SafeConn)
	}
	// If an app with same ID exists, close it
	if old, ok := session.AppConns[appID]; ok {
		old.Close()
	}
	session.AppConns[appID] = safeConn
	session.Active = true
	session.Mutex.Unlock()

	log.Printf("App %s paired with Agent %s", appID, clientID)

	// Keep-Alive: Send Pings to the mobile app every 2s to detect drops fast
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	// Start dedicated write loop
	go safeConn.StartWriteLoop()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			session.Mutex.Lock()
			if session.AppConns[appID] != safeConn {
				session.Mutex.Unlock()
				return
			}
			// Use native ping - bypasses MsgChan
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				session.Mutex.Unlock()
				log.Printf("Relay: Ping failed for app %s, cleaning up", appID)
				e.cleanupApp(clientID, appID, safeConn)
				return
			}
			session.Mutex.Unlock()
		}
	}()

	// Listen for messages from App (Blocks)
	e.proxyApp(clientID, appID, safeConn)
}

func (e *RelayEngine) proxyAgent(clientID string, agentConn *SafeConn) {
	e.Mutex.Lock()
	session, ok := e.Sessions[clientID]
	e.Mutex.Unlock()

	if !ok {
		return
	}

	for {
		// Verify this is still the active agent connection
		session.Mutex.Lock()
		if session.AgentConn != agentConn {
			session.Mutex.Unlock()
			return
		}
		session.Mutex.Unlock()

		mt, msg, err := agentConn.ReadMessage()
		if err != nil {
			log.Printf("Relay: Error reading from agent (%s): %v", clientID, err)
			e.cleanupAgent(clientID, agentConn)
			return
		}

		// Broadcast to all apps
		session.Mutex.Lock()
		for id, appConn := range session.AppConns {
			err = appConn.WriteMessage(mt, msg)
			if err != nil {
				log.Printf("Relay: Error writing to app %s (%s): %v", id, clientID, err)
				// We don't cleanup here, let the app's own proxy loop handle its disconnect
			}
		}
		session.Mutex.Unlock()
	}
}

func (e *RelayEngine) proxyApp(clientID string, appID string, appConn *SafeConn) {
	e.Mutex.Lock()
	session, ok := e.Sessions[clientID]
	e.Mutex.Unlock()

	if !ok {
		return
	}

	for {
		// Verify this is still the active connection for this appID
		session.Mutex.Lock()
		if session.AppConns[appID] != appConn {
			session.Mutex.Unlock()
			return
		}
		agentConn := session.AgentConn
		session.Mutex.Unlock()

		mt, msg, err := appConn.ReadMessage()
		if err != nil {
			log.Printf("Relay: Error reading from app %s (%s): %v", appID, clientID, err)
			e.cleanupApp(clientID, appID, appConn)
			return
		}

		// Skip proxying JSON pings to agent to avoid unnecessary noise
		if mt == websocket.TextMessage && strings.Contains(string(msg), "\"type\":\"ping\"") {
			appConn.WriteJSON(map[string]string{"type": "pong"})
			continue
		}

		if agentConn != nil {
			err = agentConn.WriteMessage(mt, msg)
			if err != nil {
				log.Printf("Relay: Error writing from app %s to agent (%s): %v", appID, clientID, err)
			}
		} else {
			log.Printf("Relay: No agent for message from app %s (%s)", appID, clientID)
		}
	}
}

func (e *RelayEngine) cleanupAgent(clientID string, failedConn *SafeConn) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	session, ok := e.Sessions[clientID]
	if !ok {
		return
	}

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	if session.AgentConn == failedConn {
		// Close all apps if the agent is gone
		for id, appConn := range session.AppConns {
			appConn.Close()
			delete(session.AppConns, id)
		}
		session.AgentConn = nil
		session.Active = false
		delete(e.Sessions, clientID)
		go markMachineOffline(clientID)
		log.Printf("Agent %s disconnected, relay session ended", clientID)
	}
}

func (e *RelayEngine) cleanupApp(clientID string, appID string, failedConn *SafeConn) {
	e.Mutex.Lock()
	session, ok := e.Sessions[clientID]
	e.Mutex.Unlock()

	if !ok {
		return
	}

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	if session.AppConns[appID] == failedConn {
		delete(session.AppConns, appID)
		log.Printf("App %s disconnected from Agent %s", appID, clientID)

		// If no more apps are connected, we might want to update status
		if len(session.AppConns) == 0 {
			session.Active = false
			go markStreamEnded(clientID)
			log.Printf("All apps disconnected from Agent %s. Stream ended.", clientID)
		}
	}
}

func markMachineOffline(clientID string) {
	var machine models.Machine
	// Internal system task: Bypass RLS to update status
	tx := db.DB.Begin()
	tx.Exec("SET LOCAL app.is_admin = 'true'")

	if err := tx.Where("client_id = ?", clientID).First(&machine).Error; err != nil {
		tx.Rollback()
		return
	}

	now := time.Now()
	tx.Model(&machine).Update("status", models.MachineStatusOffline)
	tx.Model(&models.ShareSession{}).
		Where("machine_id = ? AND status = ?", machine.ID, models.ShareSessionStatusWaiting).
		Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusCancelled,
			"ended_at": now,
		})
	tx.Model(&models.ShareSession{}).
		Where("machine_id = ? AND status = ?", machine.ID, models.ShareSessionStatusStreaming).
		Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusEnded,
			"ended_at": now,
		})
	tx.Commit()
}

func markStreamEnded(clientID string) {
	var machine models.Machine
	// Internal system task: Bypass RLS to update status
	tx := db.DB.Begin()
	tx.Exec("SET LOCAL app.is_admin = 'true'")

	if err := tx.Where("client_id = ?", clientID).First(&machine).Error; err != nil {
		tx.Rollback()
		return
	}

	var shareSession models.ShareSession
	err := tx.Where("machine_id = ? AND status = ?", machine.ID, models.ShareSessionStatusStreaming).
		Order("created_at DESC").
		First(&shareSession).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Relay: failed to load active share session for %s: %v", clientID, err)
		}
		tx.Rollback()
		return
	}

	now := time.Now()
	if err := tx.Model(&shareSession).Updates(map[string]interface{}{
		"status":   models.ShareSessionStatusEnded,
		"ended_at": now,
	}).Error; err != nil {
		log.Printf("Relay: failed to end share session for %s: %v", clientID, err)
		tx.Rollback()
		return
	}

	if err := tx.Model(&machine).Update("status", models.MachineStatusOnline).Error; err != nil {
		log.Printf("Relay: failed to reset machine status for %s: %v", clientID, err)
	}
	tx.Commit()
}
