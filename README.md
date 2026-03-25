# TermViewer

TermViewer is a terminal streaming platform for controlling a PC shell from a phone.

Today the repository contains two closely related product tracks:

- **LAN mode**: a working Go agent plus Flutter client for local-network terminal sharing
- **Public server mode**: an in-progress relay stack with a Go backend, a Next.js dashboard, and Keycloak-based identity

The LAN product is already usable. The public server product now has the core session lifecycle, secure QR behavior, mobile public access, and admin approval flow, but it still needs more production polish and hardening.

## Repository layout

- `agent/`: Go host agent, PTY/session handling, LAN websocket server
- `app/`: Flutter mobile client
- `server/backend/`: Go backend for the public relay, machine management, and onboarding
- `server/frontend/`: Next.js dashboard for the public/server mode
- `server/infrastructure/`: local infrastructure bootstrap such as Traefik, Postgres, and Keycloak
- `docs/`: architecture, API, security, roadmap, and remote-access planning docs

## Current status

### LAN mode

Implemented today:

- mDNS discovery
- TLS with certificate pinning / TOFU
- HMAC-based agent authentication
- session listing and terminal interaction
- file transfer
- clipboard sync
- terminal recording
- system HUD

### Public server mode

Implemented foundations already present in code:

- user registration
- admin approval endpoint
- activation endpoint
- machine registration with one-time `client_id` + `client_secret`
- Keycloak-backed web login
- machine heartbeats and presence tracking
- share-session lifecycle with short-lived session tokens
- relay websocket endpoints for machines and share sessions
- dashboard QR/share flow
- secure admin approval console with Keycloak admin-role enforcement
- pending-approval and activation pages
- Flutter public-mode profile, OIDC, QR, and machine-list flow

Still incomplete by design:

- OS-level deep-link handling outside the Flutter app
- further UI polish across the web and mobile public flows
- broader production hardening around deployment, observability, and admin/audit ergonomics

## Quick start

### LAN mode

Build the agent:

```bash
make build
```

Run the Flutter app:

```bash
cd app
flutter pub get
flutter run
```

See `GET_STARTED.md` for the current LAN-focused workflow.

### Public server prototype

Start the supporting services:

```bash
cd server/infrastructure
cp .env.template .env
docker compose up -d
```

Run the backend:

```bash
cd server/backend
cp .env.template .env
go run .
```

Run the frontend:

```bash
cd server/frontend
cp .env.template .env.local
npm install
npm run dev
```

Notes:

- generate a strong `NEXTAUTH_SECRET` before starting the frontend
- the backend supports either `DATABASE_URL` or the `DB_*` variables from `server/backend/.env.template`
- `server/infrastructure/.env.template`, `server/backend/.env.template`, and `server/frontend/.env.template` are the committed starting points for the stack

## Documentation

- `GET_STARTED.md`: local development and LAN usage
- `docs/ARCHITECTURE.md`: current LAN architecture
- `docs/API_SPEC.md`: agent websocket protocol
- `docs/SECURITY.md`: LAN security model and operational guidance
- `docs/REMOTE_ACCESS_ARCHITECTURE.md`: target production architecture for internet-facing mode
- `docs/REMOTE_SESSION_FLOW.md`: canonical user, machine, share-session, QR, and mobile OIDC flow
- `docs/ROADMAP.md`: high-level delivery roadmap

## Important remote-access rule

Machine credentials and share-session tokens are different things:

- `client_id` + `client_secret` are **machine enrollment credentials**
- QR codes must contain a **short-lived, single-use share-session token**

The dashboard must never expose machine secrets after setup.
