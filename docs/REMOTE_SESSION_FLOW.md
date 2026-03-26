# TermViewer Remote Session Flow

This document is the canonical product-flow reference for TermViewer public/server mode.

Use it together with `REMOTE_ACCESS_ARCHITECTURE.md`:

- `REMOTE_ACCESS_ARCHITECTURE.md` explains the infrastructure and system shape
- `REMOTE_SESSION_FLOW.md` explains the user flow, machine flow, session flow, QR rules, missing pages, and Flutter public-mode requirements

## 1. Why this document exists

The repository contains a functional remote-access stack. Most of the flows described here are implemented end to end.

The current implementation status:

1. QR scanning is implemented in the Flutter app.
2. OIDC login with Authorization Code + PKCE is implemented in Flutter.
3. Machine list and device selection flow is implemented.
4. Admin approval and activation pages exist in the web UI.
5. The backend relay, session orchestration, and machine lifecycle are operational.

The remaining gaps are:

1. OS-level deep-link handling (e.g., `termviewer://` scheme registration) is not yet wired on all platforms.
2. Further UI polish and edge-case handling across web and mobile surfaces.
3. Production hardening (rate limiting tuning, observability dashboards, load testing).

This document defines the target behavior and serves as the canonical reference for the product flow.

## 2. Core principles

### 2.1 Separate the three identities

TermViewer public mode must clearly separate:

- **User identity**
  - who owns the account
  - authenticated with Keycloak / OIDC
- **Machine identity**
  - which PC agent belongs to that user
  - authenticated with one-time `client_id` + `client_secret`
- **Share-session identity**
  - which active remote terminal share is available right now
  - authenticated with a short-lived share token

Those three identities must never be collapsed into one credential.

### 2.2 QR is for a share session, not for machine enrollment

QR codes must never contain:

- `client_secret`
- raw machine enrollment credentials
- long-lived credentials of any kind

QR codes are only for **joining an active share session**.

### 2.3 LAN mode stays first-class

The existing LAN experience remains supported:

- mDNS discovery
- manual host/port fallback
- direct WSS connection to the agent

Public-server mode is additive, not a replacement.

### 2.4 Flutter public auth follows OAuth2 best practices

The Flutter app should use:

- OIDC discovery
- OAuth2 Authorization Code + PKCE
- system browser / secure auth session
- secure token storage

It should not embed a custom username/password form for Keycloak.

## 3. Actors

- **Admin**
  - approves new user registrations
- **End user**
  - owns one or more machines
- **Agent machine**
  - the PC running the TermViewer host agent
- **Backend**
  - machine registry, session orchestration, relay authorization
- **Keycloak**
  - user identity provider
- **Flutter app**
  - terminal viewer and mobile control surface

## 4. Required web and mobile surfaces

### 4.1 Web pages required

The public web product needs at least these screens:

- landing/login page
- registration request page
- pending-approval page
- activation result page
- user dashboard
- machine details / machine share state
- admin pending-users page
- account/profile/settings page

Today the repository only has:

- `/`
- `/register`
- `/dashboard`

So the web product is currently incomplete.

### 4.2 Mobile screens required

The Flutter app needs these public-mode surfaces:

- LAN/Public mode entry point
- server profile list
- add/edit server profile screen
- OIDC sign-in flow
- machine list / device list
- QR scanner
- deep-link handler
- remote session launch flow

Today the repository is still LAN-first and does not provide these screens yet.

## 5. Credentials and tokens

| Type | Used by | Lifetime | Where shown | Purpose |
| --- | --- | --- | --- | --- |
| `client_id` | Agent | long-lived | dashboard once | machine identifier |
| `client_secret` | Agent | long-lived, rotatable | dashboard once | machine enrollment secret |
| user access token | Web/Mobile | short-lived | never manually shown | authenticated user API access |
| user refresh token | Web/Mobile | longer-lived | never manually shown | session renewal |
| share-session token | Mobile QR/direct join | very short-lived, single-use | QR only while waiting | join one active share |

Rules:

- `client_secret` is stored hashed on the backend
- `client_secret` is never displayed again after setup
- share-session tokens are single-use and expire quickly
- share-session tokens should be stored hashed if persisted server-side

## 6. Canonical product flow

### 6.1 User registration and approval

1. User opens the public web frontend.
2. User submits:
   - email
   - username
   - password
3. Backend creates the Keycloak user in a disabled state.
4. Backend stores the local user row with status `pending_approval`.
5. Admin sees the user in a pending-approval screen.
6. Admin approves the user.
7. Backend issues a single-use activation token valid for up to 24 hours.
8. User receives an activation email.
9. User opens the activation link.
10. Backend enables the Keycloak user and marks the local user as active.

### 6.2 User login

After activation:

- web login uses OIDC
- mobile login uses OIDC with Authorization Code + PKCE

If anti-enumeration is implemented, the frontend may still keep a lightweight email pre-check, but credential handling should remain with Keycloak.

### 6.3 Machine enrollment

From the dashboard:

1. Authenticated user creates a new machine entry.
2. Backend generates:
   - `client_id`
   - `client_secret`
3. Backend stores only the hashed secret.
4. Dashboard shows the secret once.
5. User configures the agent on the PC with those credentials.

This step is **machine enrollment**, not session sharing.

### 6.4 Agent online presence

When the PC agent starts:

1. Agent authenticates with machine credentials.
2. Agent establishes a persistent backend connection.
3. Agent sends heartbeats.
4. Backend updates:
   - `last_seen`
   - machine status

Machine status should be one of:

- `offline`
- `online`
- `waiting`
- `streaming`

`online` means the machine is reachable, but there is no active share token yet.

### 6.5 Share-session creation

The machine should only expose a QR code when it is **actively waiting to share**.

The intended flow is:

1. The PC agent is online.
2. The local user starts a share or marks the machine as ready.
3. Agent asks the backend to create a new share session.
4. Backend creates:
   - a share-session row
   - a high-entropy share token
   - a short expiry window
5. Machine status becomes `waiting`.
6. The dashboard shows the QR code near that machine.

When the share ends, expires, or is consumed:

- the QR code disappears
- the machine returns to `online` or `offline`

### 6.6 QR code format

The QR code should contain a deep link for one active share session.

Recommended format:

```text
termviewer://connect?server=https%3A%2F%2Fapi.termviewer.example&session_token=XYZ&refresh_token=ABC
```

Required properties:

- includes the target server base URL
- includes a short-lived share-session token
- includes a refresh token the phone can use to renew the active stream before expiry
- single-use
- scoped to one machine and one share session
- expires quickly

Optional fields may include:

- `expires_at`
- `share_id`
- `display_name`

The QR code must **not** include:

- `client_secret`
- `client_id`
- Keycloak user tokens

### 6.7 Mobile QR flow

1. User opens the Flutter app scanner.
2. App scans the QR code.
3. App extracts:
   - server base URL
   - share-session token
   - share-session refresh token
4. App validates the deep-link format.
5. App redeems the share-session token with the backend.
6. Backend validates:
   - token exists
   - token not expired
   - token not already consumed
7. Backend binds the mobile connection to that waiting share.
8. Mobile opens the terminal session.
9. Machine status becomes `streaming`.
10. Before the active share token reaches its final minute, the mobile app calls the backend refresh endpoint with the refresh token and rotates to the new session token for future reconnects.

If the Flutter app does not yet know that server:

- it may temporarily trust the server from the QR flow for that single connection
- or it may offer to save a permanent server profile

### 6.8 Mobile authenticated dashboard flow

The mobile app should also support a full authenticated public mode.

That flow is:

1. User creates or selects a server profile.
2. User configures:
   - server API/base URL
   - OIDC issuer URL
   - OIDC client ID
   - redirect URI / app callback scheme
   - optional scopes
3. App performs OIDC discovery.
4. App signs in with Authorization Code + PKCE.
5. App stores tokens securely.
6. App loads the user’s machines.
7. User sees machine state:
   - offline
   - online
   - waiting
   - streaming
8. User can choose a machine to connect to.

Possible behaviors:

- if a machine is already `waiting`, join the waiting share
- if a machine is `online`, request a new share session
- if a machine is `streaming`, show the active session state or offer takeover rules later
- if a machine is `offline`, show it as unavailable

## 7. Flutter OIDC requirements

The Flutter app should support one or more configurable server profiles.

Each profile should at minimum store:

- profile name
- API base URL
- OIDC issuer URL
- OIDC client ID
- redirect URI / callback scheme
- default scopes

Recommended auth behavior:

- use OIDC discovery from the issuer URL
- use Authorization Code + PKCE
- prefer a system browser / ASWebAuthenticationSession / Custom Tabs flow
- store refresh/access tokens in secure storage
- handle logout per server profile

The current Flutter app does not yet implement this.

## 8. Relay binding rule

The relay should be bound to a **share session**, not directly to a `client_id`.

Current prototype behavior in the repository pairs the mobile app directly with a machine by `clientID`. That is not the final design.

Correct target behavior:

- agent connects as a machine
- mobile connects as a user or share-session consumer
- backend authorizes the mobile side against a specific waiting share session
- relay routes traffic by `share_session_id`

This ensures:

- session isolation
- correct QR semantics
- correct takeover/expiry rules
- no direct exposure of machine enrollment identity to the mobile app

## 9. State machines

### 9.1 User state

```text
requested -> pending_approval -> approved_pending_activation -> active
```

Optional future states:

```text
suspended
disabled
```

### 9.2 Machine state

```text
created -> configured -> offline -> online -> waiting -> streaming
```

With transitions back to:

```text
online
offline
```

### 9.3 Share-session state

```text
created -> waiting -> consumed -> streaming -> ended
```

Additional terminal states:

```text
expired
cancelled
failed
```

## 10. API and UI implications for the current repository

This flow implies the next implementation phases must add at least:

- share-session entities in the backend
- machine heartbeat and state tracking
- QR generation from share tokens instead of machine credentials
- web pages for activation, pending approval, and admin approvals
- Flutter QR scanning
- Flutter deep-link handling
- Flutter configurable OIDC server profiles
- Flutter OIDC login with PKCE
- authenticated machine listing in the app

## 11. Canonical decisions

These decisions should be treated as locked unless intentionally revised:

- QR codes use short-lived share-session tokens only
- machine credentials are setup-only and never re-displayed
- public Flutter auth uses OIDC with Authorization Code + PKCE
- the app must support configurable OIDC/server profiles
- LAN mode remains supported in parallel with public mode
