# 📄 TermViewer Remote Access Architecture (Production-Ready)

## 1. 🎯 Objective

Design a secure, scalable, and production-grade remote terminal streaming platform that:

* Enables **secure remote terminal access over the internet**
* Uses **industry-standard authentication (OIDC)** via Keycloak
* Ensures **zero exposure of long-lived credentials**
* Provides both:

  * QR-based instant connection
  * Dashboard-driven device access (TeamViewer-like)

For the canonical product flow, missing page inventory, QR rules, and Flutter public-mode requirements, see `REMOTE_SESSION_FLOW.md`.

---

# 2. 🧱 System Architecture

## 2.1 Core Components

### Identity Layer

* **Keycloak**

  * OIDC Provider
  * User lifecycle (registration, activation, login)
  * Token issuance (JWT)

---

### Edge / Gateway Layer

* **Traefik**

  * TLS termination
  * Routing (API, WS, frontend)
  * Middleware (rate limiting, headers)
  * Let’s Encrypt automation

---

### Application Layer

#### Backend (Go)

* REST API (Gin / Fiber idiomatic)
* WebSocket relay (high-performance)
* Session orchestration
* Device lifecycle management

#### Frontend (Web)

* Dashboard UI (React / Angular)
* Device management
* QR visualization
* Admin panel

#### Mobile App (Flutter)

* QR scanner
* Terminal viewer
* Optional dashboard access via OIDC

---

### Data Layer

* **PostgreSQL**

  * relational integrity
  * transactional consistency
  * indexing for real-time queries

---

### Agent Layer

* Go daemon running on client machine
* Maintains persistent connection to backend
* Streams terminal via PTY

---

# 3. 🔐 Identity & Access Management (OIDC)

## 3.1 Keycloak Configuration (Best Practice)

### Realm Design

```
realm: termviewer
```

### Clients

#### 1. Frontend Client

```
type: public
flow: Authorization Code + PKCE
```

#### 2. Mobile Client

```
type: public
flow: Authorization Code + PKCE
```

#### 3. Backend Client

```
type: confidential
flow: service account (client credentials)
```

#### 4. Agent Client

```
type: confidential
flow: client credentials
```

---

## 3.2 Token Strategy

### Access Token (JWT)

* short-lived (5–15 min)
* contains:

  ```
  sub (user_id)
  scope
  roles
  ```

### Refresh Token

* used by frontend/mobile only

### Agent Token

* obtained via client credentials
* rotated periodically

---

# 4. 👤 User Lifecycle (Strict & Secure)

## 4.1 Registration Flow

1. User submits registration request (frontend → backend)
2. Backend:

   * creates user in Keycloak (disabled)
   * stores metadata in DB
3. Status:

```
PENDING_APPROVAL
```

---

## 4.2 Admin Approval

Admin (via dashboard):

* approves user

Backend:

* enables user in Keycloak
* generates activation token

---

## 4.3 Activation Flow

Email contains:

```
/activate?token=XYZ
```

Constraints:

* expires in 24h
* single-use

---

## 4.4 Login Flow (Anti-enumeration)

Step 1:

```
POST /auth/check-email
→ always 200
```

Step 2:

```
OIDC redirect (Keycloak)
```

Handled entirely by Keycloak → avoids custom auth vulnerabilities.

---

# 5. 💻 Device Management

## 5.1 Device Model

```sql
devices (
  id UUID PK,
  user_id UUID FK,
  name TEXT,
  client_id TEXT UNIQUE,
  client_secret_hash TEXT,
  status TEXT,
  created_at TIMESTAMP
)
```

---

## 5.2 Device Registration

User creates device → backend:

* generates:

  ```
  client_id
  client_secret
  ```
* stores:

  * hashed secret (bcrypt/argon2)
* returns secret **ONCE**

---

## 5.3 Device Authentication (Agent)

Agent:

```
POST /auth/agent
(client_id + client_secret)
```

Backend:

* validates
* issues JWT

---

# 6. 🤖 Agent Runtime Behavior

## 6.1 Persistent Connection

Agent establishes:

```
WSS → /ws/agent
```

Includes:

```
Authorization: Bearer <JWT>
```

---

## 6.2 Heartbeat System

Every N seconds:

```
PING → backend
```

Backend updates:

```
last_seen
status = ONLINE
```

Failure:

```
status = OFFLINE
```

---

## 6.3 Terminal Stream

* PTY → binary stream
* compressed (zstd recommended)
* multiplexed channels:

  ```
  STDOUT
  STDERR
  CONTROL
  ```

---

# 7. 📡 Session Management (CRITICAL DESIGN)

## 7.1 Session Entity

```sql
sessions (
  id UUID PK,
  device_id UUID,
  status TEXT,
  session_token TEXT UNIQUE,
  expires_at TIMESTAMP,
  created_at TIMESTAMP
)
```

---

## 7.2 Session Creation

Agent:

```
POST /sessions
```

Backend generates:

```
session_token (random, high entropy)
TTL: 60–120s
status: WAITING
```

---

## 7.3 QR Code Design (Best Practice)

QR contains:

```
termviewer://connect?server=https%3A%2F%2Fapi.termviewer.example&session_token=XYZ
```

### Security Rules

* single-use
* short-lived
* scoped to one session
* NOT reusable

---

# 8. 📲 Mobile Connection Flows

## 8.1 QR-Based Instant Connection

1. Scan QR
2. Extract server + token
3. Call:

```
POST /sessions/connect
```

Backend:

* validates token
* binds mobile to session
* upgrades to WebSocket

---

## 8.2 Dashboard-Based Connection

Mobile authenticates via:

* OIDC (Keycloak) using a configured server profile

Fetch:

```
GET /devices
```

User selects device:

```
POST /sessions/request
```

Backend:

* notifies agent (via WS)
* agent accepts (optional)
* session created

---

# 9. 🔁 Relay Architecture (WebSocket Core)

## 9.1 Connection Topology

```
Agent <====> Backend Relay <====> Mobile
```

---

## 9.2 Authentication Rules

### Agent

* JWT (client credentials)

### Mobile

* either:

  * session_token
  * OR user JWT

---

## 9.3 Session Isolation (MANDATORY)

Each session:

* dedicated channel
* strict mapping:

  ```
  session_id → agent_socket + mobile_socket
  ```

No cross-session access allowed.

---

## 9.4 Message Routing

```json
{
  "type": "data",
  "channel": "stdout",
  "payload": "..."
}
```

---

# 10. 🔒 Security Model

## 10.1 Transport Security

* TLS everywhere (via Traefik)

---

## 10.2 Secret Handling

* client_secret → hashed
* never logged
* never re-displayed

---

## 10.3 Rate Limiting

* login attempts
* session creation
* token validation

---

## 10.4 Replay Protection

* session_token → single-use
* expires quickly

---

## 10.5 Optional E2E Encryption (Advanced)

* agent generates ephemeral keypair
* mobile receives public key
* relay passes encrypted payloads only

---

# 11. 📊 Device State Machine

```
OFFLINE → ONLINE → STREAMING
           ↓
        DISCONNECTED
```

---

# 12. 🚀 Deployment Strategy

## 12.1 Infrastructure (Docker Compose Split)

* traefik/
* keycloak/
* postgres/
* backend/
* frontend/

Each independently deployable.

---

## 12.2 Traefik Routing Example

```
api.termviewer.io → backend
app.termviewer.io → frontend
auth.termviewer.io → keycloak
ws.termviewer.io → websocket relay
```

---

## 12.3 Observability

* structured logs (JSON)
* metrics (Prometheus)
* tracing (OpenTelemetry)

---

# 13. 📈 Scaling Considerations

* stateless backend (horizontal scaling)
* Redis (optional) for:

  * session cache
  * pub/sub for WS routing
* DB indexing:

  ```
  sessions(session_token)
  devices(user_id)
  ```

---

# 14. ✅ Key Best Practices Summary

* ❌ Never expose `client_secret` beyond setup
* ✅ Use ephemeral `session_token` for QR
* ✅ Use OIDC (Keycloak) for ALL users
* ✅ Separate:

  * user auth
  * device auth
* ✅ Enforce session isolation
* ✅ Keep relay stateless if possible
* ✅ Use short-lived tokens everywhere

---

# 15. 🔮 Future Enhancements

* Device trust model (whitelisting)
* Multi-user sessions
* Session recording encryption
* Zero-trust networking layer (WireGuard-like)

---
