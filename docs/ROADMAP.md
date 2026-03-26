# TermViewer Roadmap: Towards Production-Ready

This document tracks the evolution of TermViewer from a prototype to a professional-grade terminal streaming platform.

## 1. Production-Readiness & Reliability
- [x] **Structured Logging:** Migrate Go Agent to `log/slog` for JSON-structured logs.
- [x] **Stream Compression:** Implement `zlib` or `zstd` compression for the WebSocket binary stream to reduce latency and data usage.
- [x] **Robust Reconnection:** Implement a "Mosh-like" sequence numbering system to handle Wi-Fi/Cellular handovers seamlessly.
- [ ] **Process Monitoring:** Automated health checks and auto-restart capability for the Agent daemon.

## 2. Security Hardening
- [x] **Certificate Pinning (TOFU):** Update the Mobile App to store and verify the Agent's TLS certificate fingerprint on first connection.
- [x] **Rate Limiting:** Implement brute-force protection to temporarily block IPs after multiple failed HMAC attempts.
- [ ] **Sandboxing:** Add a configuration option to restrict the Agent to specific directories or run it within a container.

## 3. Professional Features
- [x] **Bidirectional Clipboard Sync:** Synchronize the host and mobile clipboards automatically over a secure control channel.
- [x] **Integrated File Transfer:** A UI-driven browser in the mobile app to upload/download files to/from the host.
- [x] **Bidirectional File Transfer:** Support uploading files/folders from phone to PC.
- [x] **Custom Macros & Keybar:** A user-customizable toolbar for modifier keys and frequently used CLI commands.
- [x] **Terminal Recording:** Support for recording sessions in `asciinema` (.cast) format.
- [x] **System HUD:** A small overlay showing real-time Host CPU, RAM, and Network usage.


## 4. UI/UX & Aesthetics
- [x] **Theme Engine:** Support for standard terminal color schemes (Dracula, Nord, Solarized, etc.).
- [ ] **Advanced Gestures:** Multi-touch support for font scaling, fluid scrolling, and a cursor magnifier.
- [ ] **Haptic Feedback:** Optional haptic "clicks" for virtual keyboard and terminal events.

## 5. Packaging & Distribution
- [x] **Debian Packaging (.DEB):**
    - [x] Create a `systemd` unit file for the TermViewer Agent.
    - [ ] Script the `postinst` and `prerm` steps for certificate generation and user management.
    - [x] Bundle all dependencies into a single, installable `.deb` package for Ubuntu/Debian systems.
- [ ] **Automated CI/CD:** GitHub Actions to build and release Agent binaries for multiple architectures (ARM64, x64) and App bundles (APK, IPA).
- [ ] **Desktop Integration:** Background tray icon for the Agent on Windows, macOS, and Linux (GNOME/KDE).

## 6. Enterprise-Level Intermediate Server (Remote Access)
- [x] **Architecture & Infrastructure:**
    - [x] Traefik v3 (latest) as reverse proxy / API gateway with dynamic and static configuration.
    - [x] Keycloak (latest) as the primary Identity Provider (IdP) in production mode (HTTPS).
    - [x] PostgreSQL for robust database storage.
    - [x] GORM for database interactions in the backend.
    - [x] Separate Docker Compose deployments for each component (Traefik, Keycloak, Postgres, Backend, Frontend) for scalability.
- [x] **Authentication & Onboarding Flow:**
    - [x] Real OIDC integration and secure protocols.
    - [x] User requests account creation (email, username, password).
    - [x] Admin console for manual account approval.
    - [x] Automated email sender upon approval containing an activation link (max 24h expiry).
    - [x] Secure two-step login flow: Step 1 (email check, quietly blocks brute-force if no account found), Step 2 (password verification).
- [x] **Dashboard, Machine Enrollment & Session Sharing:**
    - [x] User dashboard displaying account profile and registered machines.
    - [x] Activation result, pending approval, and admin review pages for the web UI.
    - [x] Ability to generate a unique `ClientID` and `ClientSecret` for each agent machine, with the secret shown only once.
    - [x] Agent authenticates with the intermediate server using the generated `ClientID` and `ClientSecret`.
    - [x] Track machine state (`offline`, `online`, `waiting`, `streaming`) and heartbeat/last-seen status.
    - [x] Create short-lived share sessions for active remote terminal sharing.
    - [x] Generate dashboard QR codes from short-lived share-session tokens, never from machine credentials.
- [x] **Mobile App Public Mode:**
    - [x] QR scan and deep-link handling for public-server share sessions.
    - [x] Configurable server profiles in Flutter (API URL, OIDC issuer URL, client ID, redirect URI/scopes).
    - [x] Built-in OIDC login in Flutter using OAuth2 Authorization Code + PKCE.
    - [x] Authenticated device list and machine selection from the phone.
- [ ] **Secure Backend Relay & Tunneling:**
    - [x] Implementation of a high-performance backend relay to proxy traffic between the agent and phone app.
    - [x] Mutual authentication and authorization check: the relay only permits connections where both the agent and the mobile app have a valid session verified against Keycloak.
    - [x] Session isolation: each remote terminal session is strictly sandboxed within the relay to prevent cross-session data leakage.
    - [x] Secure signaling: a dedicated control channel to manage connection handshakes and heartbeats.
    - [ ] Support for end-to-end encryption (TLS passthrough or relay-re-encryption) to ensure the server cannot inspect the raw terminal stream.
