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
- [ ] **Debian Packaging (.DEB):**
    - [ ] Create a `systemd` unit file for the TermViewer Agent.
    - [ ] Script the `postinst` and `prerm` steps for certificate generation and user management.
    - [ ] Bundle all dependencies into a single, installable `.deb` package for Ubuntu/Debian systems.
- [ ] **Automated CI/CD:** GitHub Actions to build and release Agent binaries for multiple architectures (ARM64, x64) and App bundles (APK, IPA).
- [ ] **Desktop Integration:** Background tray icon for the Agent on Windows, macOS, and Linux (GNOME/KDE).

## 6. Enterprise-Level Intermediate Server (Remote Access)
- [ ] **Architecture & Infrastructure:**
    - [ ] Traefik v3 (latest) as reverse proxy / API gateway with dynamic and static configuration.
    - [ ] Keycloak (latest) as the primary Identity Provider (IdP) in production mode (HTTPS).
    - [ ] PostgreSQL for robust database storage.
    - [ ] GORM for database interactions in the backend.
    - [ ] Separate Docker Compose deployments for each component (Traefik, Keycloak, Postgres, Backend, Frontend) for scalability.
- [ ] **Authentication & Onboarding Flow:**
    - [ ] Real OIDC integration and secure protocols.
    - [ ] User requests account creation (email, username, password).
    - [ ] Admin console for manual account approval.
    - [ ] Automated email sender upon approval containing an activation link (max 24h expiry).
    - [ ] Secure two-step login flow: Step 1 (email check, quietly blocks brute-force if no account found), Step 2 (password verification).
- [ ] **Dashboard, Machine Enrollment & Session Sharing:**
    - [ ] User dashboard displaying account profile and registered machines.
    - [ ] Activation result, pending approval, and admin review pages for the web UI.
    - [ ] Ability to generate a unique `ClientID` and `ClientSecret` for each agent machine, with the secret shown only once.
    - [ ] Agent authenticates with the intermediate server using the generated `ClientID` and `ClientSecret`.
    - [ ] Track machine state (`offline`, `online`, `waiting`, `streaming`) and heartbeat/last-seen status.
    - [ ] Create short-lived share sessions for active remote terminal sharing.
    - [ ] Generate dashboard QR codes from short-lived share-session tokens, never from machine credentials.
- [ ] **Mobile App Public Mode:**
    - [ ] QR scan and deep-link handling for public-server share sessions.
    - [ ] Configurable server profiles in Flutter (API URL, OIDC issuer URL, client ID, redirect URI/scopes).
    - [ ] Built-in OIDC login in Flutter using OAuth2 Authorization Code + PKCE.
    - [ ] Authenticated device list and TeamViewer-like machine selection from the phone.
- [ ] **Secure Backend Relay & Tunneling:**
    - [ ] Implementation of a high-performance backend relay to proxy traffic between the agent and phone app.
    - [ ] Mutual authentication and authorization check: the relay only permits connections where both the agent and the mobile app have a valid session verified against Keycloak.
    - [ ] Session isolation: each remote terminal session is strictly sandboxed within the relay to prevent cross-session data leakage.
    - [ ] Secure signaling: a dedicated control channel to manage connection handshakes and heartbeats.
    - [ ] Support for end-to-end encryption (TLS passthrough or relay-re-encryption) to ensure the server cannot inspect the raw terminal stream.
