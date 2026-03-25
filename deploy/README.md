# TermViewer Deployment Infrastructure

This directory contains production-ready Docker Compose configurations for deploying TermViewer (Frontend, Backend, PostgreSQL, Keycloak) securely behind Traefik.

## Quick Start

Run the interactive setup script to generate a complete `.env` file with auto-generated passwords and secrets:

```bash
cd deploy
./setup.sh
```

The script will:
- Ask you to choose **testing** or **production**
- Walk you through domain, database, Keycloak, NextAuth, SMTP, and TLS configuration
- Auto-generate strong passwords and secrets (with option to override)
- Enforce minimum password lengths and reject weak defaults in production
- Write a `.env` file with `chmod 600` permissions

## Environments

1. **`testing`**: Uses Traefik with internally generated **self-signed certificates**. It mimics the production environment perfectly but does not require a real domain or ACME challenges.
2. **`production`**: Uses Traefik with **Let's Encrypt (ACME)** to automatically issue and renew trusted SSL certificates for your external domains.

## General Architecture
- **Traefik**: Acts as the edge router, terminating SSL/TLS on port 443 and automatically redirecting port 80 to 443.
- **Frontend (Next.js)**: Runs in `standalone` mode (smallest Docker footprint). Handled by Traefik at the root `/` path.
- **Backend (Go)**: Built using an alpine container. Handled by Traefik with a `PathPrefix` for `/api`, `/admin`, `/agent`, `/machines`, and `/ws`.
- **Keycloak**: Available at `auth.yourdomain.com` (or `sso.termviewer.local` for testing).
- **Postgres**: Backs both Keycloak and the Go Backend on isolated databases.

## 1. Testing Deployment

```bash
cd deploy
./setup.sh          # choose "testing"
cd testing
docker compose up -d --build
```
- Frontend: `https://termviewer.local`
- Backend API: `https://termviewer.local/api`
- Keycloak: `https://sso.termviewer.local`

## 2. Production Deployment

```bash
cd deploy
./setup.sh          # choose "production"
cd production
docker compose up -d --build
```

Make sure your DNS A records for your domain, `auth.<domain>`, and `logs.<domain>` point to your server's public IP.

**Security Notes:**
- All traffic between Traefik and internal services runs securely on a dedicated, isolated Docker network (`termviewer_net`).
- Only ports 80 and 443 are exposed to the host machine.
- The `acme.json` file securely stores your Let's Encrypt certificates and has `chmod 600` permissions.
- The `.env` file contains all secrets – never commit it to version control.
