# Security Design: TermViewer

Security is a primary concern for TermViewer, as it involves remote execution and control of a computer system over a network.

## 1. Transport Layer Security (TLS)
All communications between the TermViewer Mobile App and the TermViewer Agent are encrypted using TLS 1.3.

*   **WSS (WebSocket Secure):** No unencrypted WebSocket connections (WS) are allowed. The server exclusively listens for WSS requests.
*   **Certificate Management:**
    *   The TermViewer Agent generates a self-signed ECDSA P-256 certificate on its first run if one is not found.
    *   **Local Trust:** The mobile app is configured to trust these self-signed certificates, as the primary use case is trusted Local Area Network (LAN) communication.
*   **Encrypted Payloads:** Both control JSON messages and binary terminal I/O are encrypted at the transport layer, preventing eavesdropping on sensitive terminal data.

## 2. Authentication Mechanism
Access to the Agent's terminal is restricted by a pre-shared password mechanism.

*   **Password-Based HMAC Handshake:**
    1.  **Challenge:** The App requests authentication. The Agent generates a random 16-byte nonce.
    2.  **Response:** The App computes an **HMAC-SHA256** hash of the nonce using the user-provided password.
    3.  **Validation:** The Agent computes its own HMAC-SHA256 using the stored password and validates the response.
*   **Zero-Knowledge Authentication:** The password is **never** transmitted over the network, even in encrypted form. Only the HMAC signature is sent.
*   **UI/UX Security:** If a password is not provided, the mobile app identifies the authentication failure and prompts the user via a secure modal before retrying the connection.

## 3. Concurrency & Integrity
*   **Write Safety:** The Agent uses per-connection mutexes to ensure that WebSocket writes are atomic. This prevents data corruption or interleaved messages that could lead to terminal rendering errors or application panics.
*   **Session Isolation:** Each connection has its own lifecycle. When a client detaches or a session closes, the Agent ensures that all associated streaming goroutines are terminated and resources are reclaimed.

## 4. Operational Best Practices
*   **Privilege Level:** The TermViewer Agent should run with the lowest possible privileges. It only has access to the shell and environment of the user who started the process. It should NOT run as `root` or `Administrator`.
*   **Local Focus:** TermViewer is designed for LAN use. Exposing the TermViewer port (default 24242) to the public internet is discouraged unless through a secure VPN or tunnel (e.g., WireGuard or Tailscale).
