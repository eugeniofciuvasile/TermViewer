# Architecture: TermViewer

TermViewer is a secure, real-time terminal streaming system for controlling a PC shell from a mobile device over a Local Area Network (LAN).

## System Components

### 1. TermViewer Agent (The Host)
*   **Language:** Go (Golang)
*   **Role:** Acts as the server. It manages local shell processes (Pseudo-Terminal or PTY), handles incoming WebSocket connections, and enforces security policies.
*   **Core Logic:**
    *   **PTY Management:** Uses `github.com/creack/pty` to allocate and control native shell processes.
    *   **Session Multiplexing:** Supports multiple concurrent clients attaching to the same session. Native sessions are managed globally to allow persistence.
    *   **Concurrency Safety:** Employs per-connection mutexes to synchronize WebSocket writes, preventing race conditions and panics during high-volume data streaming.
    *   **Resource Cleanup:** Automatically detects shell exits and notifies all attached clients before cleaning up internal process state.
    *   **Terminal Recording:** Records sessions in Asciinema v2 (.cast) format, toggled by the client.
    *   **System Stats Collection:** Gathers CPU, RAM, disk, and uptime metrics and broadcasts them to connected clients every 5 seconds.
    *   **Bidirectional File Transfer:** Supports browsing, uploading, and downloading files between the client and host.
    *   **Clipboard Synchronization:** Bidirectional clipboard sync between the host and connected clients.
    *   **Keep-alive:** Ping/pong mechanism for monitoring connection health.

### 2. TermViewer Mobile App (The Client)
*   **Framework:** Flutter (Dart)
*   **Role:** Acts as the interactive UI. It discovers agents on the network, authenticates with them, and renders the terminal stream.
*   **Terminal Rendering:** Uses `xterm.dart` for high-performance ANSI/VT100 terminal emulation.
*   **Interactive UI Features:**
    *   **Discovery:** mDNS-based discovery with automatic password prompting for secure devices.
    *   **Virtual Desktop Feel:** A large virtual canvas (1200px width) allowing users to pan and zoom, suitable for complex TUI applications.
    *   **Auto-scrolling:** Intelligent viewport management that follows terminal output while maintaining zoom/pan state.
    *   **Session Picker:** Real-time list of available sessions on the host (Native TermViewer sessions or Tmux sessions).
    *   **Custom Macros & Keybar:** A configurable toolbar for modifier keys and frequently used CLI commands.
    *   **Terminal Themes:** Built-in color schemes including Dracula, Nord, Solarized Dark, and Monokai.
    *   **Bidirectional File Transfer:** Built-in file browser UI for uploading and downloading files to/from the host.
    *   **System HUD Overlay:** Real-time overlay displaying host CPU, RAM, disk, and uptime metrics.
    *   **TOFU Certificate Pinning:** Trust-on-First-Use pinning of the Agent's self-signed TLS certificate fingerprint.

## Communication Stack

| Layer | Protocol | Purpose |
| :--- | :--- | :--- |
| **Discovery** | mDNS / DNS-SD | Local network discovery without manual IP entry. |
| **Transport** | TLS 1.3 | End-to-end encryption for all traffic. |
| **Application** | WebSockets (WSS) | Persistent, full-duplex streaming. JSON for control, Binary for I/O. |
| **Synchronization**| Mutex / Atomic | Ensures thread-safe communication over the WebSocket. |
| **Keep-alive** | ping/pong | Connection health monitoring |

## Operational Lifecycle

1.  **Discovery Phase:** The Agent broadcasts its service via mDNS. The Mobile App scans and lists available agents.
2.  **Authentication:** 
    *   The App initiates a connection. 
    *   If a password is required, the App prompts the user via a modal.
    *   A HMAC-SHA256 challenge-response handshake is performed.
3.  **Session Selection:**
    *   The App requests a list of available sessions.
    *   The user can choose to attach to an existing session (Tmux or Native) or start a new one.
4.  **Streaming & Interaction:**
    *   **Output:** Raw bytes from the PTY are broadcast to all attached clients via dedicated Go channels.
    *   **Input:** Client keystrokes are sent as binary frames and written directly to the PTY's `stdin`.
    *   **File Transfer:** A side-channel for browsing, uploading, and downloading files between the client and host.
    *   **Clipboard Sync:** A dedicated channel for bidirectional clipboard sharing between the host and connected clients.
    *   **System Stats:** The Agent broadcasts host metrics (CPU, RAM, disk, uptime) to all clients every 5 seconds.
    *   **Terminal Recording:** Clients can toggle Asciinema session capture on and off.
    *   **Resizing:** Clients send window dimension changes; the Agent arbitrates the PTY size based on the **smallest common denominator** across all connected clients to ensure all viewers can display the full terminal content.
5.  **Termination:**
    *   **Client Side:** Disconnecting from the app stops the stream but can leave persistent sessions running.
    *   **Host Side:** Typing `exit` in the shell triggers an `OnExit` hook in the Agent, which notifies all clients via a `session_closed` message and cleans up the session.

> **Note:** For the public server relay architecture, see `REMOTE_ACCESS_ARCHITECTURE.md`.
