# API Specification: TermViewer WebSocket Protocol

The TermViewer WebSocket Protocol manages the communication between the Mobile App and the Agent over a secure WSS connection.

## 1. Message Format
Messages are primarily JSON-based for control and state management. Raw terminal data is sent as **Binary WebSocket Frames** to minimize overhead and avoid encoding delays.

### Generic JSON Message Structure
```json
{
  "type": "string",
  "payload": {}
}
```

## 2. Authentication Flow

### Message: `auth_request` (Client -> Agent)
Initiates the handshake.
```json
{
  "type": "auth_request",
  "payload": { "client_id": "string" }
}
```

### Message: `auth_challenge` (Agent -> Client)
Provides a unique nonce for the challenge-response.
```json
{
  "type": "auth_challenge",
  "payload": {
    "nonce": "base64_string",
    "timestamp": "iso_string"
  }
}
```

### Message: `auth_response` (Client -> Agent)
Computed using `HMAC-SHA256(nonce, password)`.
```json
{
  "type": "auth_response",
  "payload": {
    "response": "base64_hmac_sha256",
    "nonce": "base64_string"
  }
}
```

### Message: `auth_status` (Agent -> Client)
```json
{
  "type": "auth_status",
  "payload": {
    "status": "success | failed",
    "reason": "optional_error_message"
  }
}
```

## 3. Session Management

### Message: `session_list_request` (Client -> Agent)
Requests all available terminal sessions on the host.
```json
{ "type": "session_list_request" }
```

### Message: `session_list_response` (Agent -> Client)
Returns an array of sessions including Tmux and Native TermViewer sessions.
```json
{
  "type": "session_list_response",
  "payload": {
    "sessions": [
      {
        "id": "string",
        "name": "string",
        "type": "tmux | termviewer",
        "context": "string (pane preview or cwd)",
        "is_attached": true
      }
    ]
  }
}
```

## 4. Terminal Control & Streaming

### Message: `terminal_init` (Client -> Agent)
Initializes a terminal session on the agent.
```json
{
  "type": "terminal_init",
  "payload": {
    "rows": 24,
    "cols": 80,
    "command": "optional_command_to_run",
    "session_id": "optional_session_to_attach"
  }
}
```

### Raw I/O (Bi-directional)
*   **Agent -> Client:** Raw shell output sent as **Binary Frames**.
*   **Client -> Agent:** User input (keystrokes) sent as **Binary Frames**.

### Message: `terminal_resize` (Client -> Agent)
Updates the agent on the client's current viewport size.
```json
{
  "type": "terminal_resize",
  "payload": {
    "rows": 30,
    "cols": 100
  }
}
```

## 5. Termination & Errors

### Message: `session_closed` (Agent -> Client)
Sent when the terminal shell process terminates.
```json
{
  "type": "session_closed",
  "payload": { "reason": "Shell exited" }
}
```

### Message: `error` (Agent -> Client)
Sent when an operation (e.g., shell spawn) fails.
```json
{
  "type": "error",
  "payload": { "message": "string" }
}
```

## 6. File Transfer (Side-channel)

### Message: `file_list_request` (Client -> Agent)
Requests a directory listing.
```json
{
  "type": "file_list_request",
  "payload": { "path": "string" }
}
```

### Message: `file_list_response` (Agent -> Client)
```json
{
  "type": "file_list_response",
  "payload": {
    "path": "string",
    "files": [
      {
        "name": "string",
        "is_dir": true,
        "size": 1024,
        "mod_time": "iso_string"
      }
    ]
  }
}
```

### Message: `file_download_request` (Client -> Agent)
Requests a file to be sent to the client.
```json
{
  "type": "file_download_request",
  "payload": { "path": "string" }
}
```

### Message: `file_upload_start` (Client -> Agent)
Initiates a file upload to the host.
```json
{
  "type": "file_upload_start",
  "payload": {
    "transfer_id": "string",
    "filename": "string",
    "destination_path": "string"
  }
}
```

### Message: `file_data` (Bi-directional)
Used for chunked file transfer.
```json
{
  "type": "file_data",
  "payload": {
    "transfer_id": "string",
    "chunk_index": 0,
    "is_last": true,
    "data": "base64_string"
  }
}
```

## 7. Performance Monitoring

### Message: `system_stats_sync` (Agent -> Client)
Periodic update of host resource usage.
```json
{
  "type": "system_stats_sync",
  "payload": {
    "cpu_usage": 15.5,
    "ram_used_gb": 4.0,
    "ram_total_gb": 16.0,
    "disk_percent": 45.2,
    "uptime_seconds": 36000
  }
}
```

## 8. Terminal Recording

### Message: `terminal_record_toggle` (Client -> Agent)
Starts or stops recording the active terminal session.
```json
{
  "type": "terminal_record_toggle",
  "payload": { "active": true | false }
}
```

### Message: `terminal_status_response` (Agent -> Client)
Pushed by agent when status changes (like recording).
```json
{
  "type": "terminal_status_response",
  "payload": { "is_recording": true | false }
}
```
