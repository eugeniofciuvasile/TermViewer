# TermViewer Web Frontend

This directory contains the Next.js dashboard for TermViewer public/server mode.

## Current state

The frontend now covers the core onboarding and machine-sharing flows, but it still needs more product polish before it feels complete.

Pages currently present:

- `/`: marketing/landing page
- `/register`: account request form
- `/pending-approval`: post-registration holding page
- `/activate`: activation result page
- `/dashboard`: machine list and machine creation flow
- `/admin`: secure pending-user approval console for admin-role users

## Missing or incomplete pages

The current UI is missing several important screens that the public product needs:

- better account/profile/settings surfaces
- improved machine details and share-state views
- polished empty/error/loading states across the product

## QR behavior

The dashboard now follows the correct share model:

- machine credentials are shown only once at machine creation time
- QR codes appear only when a machine is actively in a waiting share state
- QR codes contain a short-lived share-session token, not machine credentials

Remaining work here is mostly UX polish around the machine/share state presentation.

See `../../docs/REMOTE_SESSION_FLOW.md` for the canonical session model.

## Auth and configuration

The current frontend uses NextAuth with Keycloak and expects these environment variables:

- `NEXT_PUBLIC_BACKEND_URL`
- `KEYCLOAK_ISSUER`
- `KEYCLOAK_CLIENT_ID`
- `KEYCLOAK_CLIENT_SECRET`
- `KEYCLOAK_ADMIN_ROLE` (defaults to `termviewer-admin`)
- `NEXTAUTH_URL`
- `NEXTAUTH_SECRET`

The backend should also allow the frontend origin through `CORS_ALLOWED_ORIGINS`.
For local development, start from the committed template:

```bash
cp .env.template .env.local
```

Then set at least:

- `NEXTAUTH_SECRET` to a strong random value
- `NEXT_PUBLIC_BACKEND_URL` to the backend origin
- `KEYCLOAK_ISSUER` / `KEYCLOAK_CLIENT_SECRET` to the matching Keycloak client settings

The secure admin console depends on the same Keycloak realm role that the backend enforces.

## Development

```bash
npm install
npm run dev
```

## Notes for the next implementation pass

The next UI-focused phase should add:

- richer machine detail views and session history
- account/profile/settings pages
- better audit-style feedback around approvals, activations, and share lifecycle events
