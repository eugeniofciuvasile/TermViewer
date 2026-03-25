# TermViewer Mobile App

This directory contains the Flutter client for TermViewer.

## What works today

The app now supports two connection modes:

- a **LAN-first** local workflow
- a **public remote-access** workflow for production servers

The LAN workflow supports:

- mDNS discovery of local agents
- manual host/port connection
- TOFU certificate trust with fingerprint mismatch protection
- terminal session selection
- terminal streaming and interaction
- file transfer
- clipboard sync
- terminal recording state
- system stats/HUD support

The public remote-access workflow supports:

- configurable server profiles
- configurable OIDC issuer/client settings
- OAuth2 Authorization Code + PKCE login via `flutter_appauth`
- in-app QR scanning for short-lived share links
- authenticated machine listing
- starting a remote share session from the device list
- joining a remote share session directly from the scanned QR

## Public-mode status

The core public flow is now implemented in the app:

1. add a server profile
2. sign in with OIDC
3. load your machines
4. connect to a machine from the authenticated list
5. or scan a share QR and join directly

The main public-mode files are:

- `lib/public_access.dart`: remote-access UI, profile editor, and QR scanner screen
- `lib/public_access_service.dart`: OIDC, API, and share-session helpers
- `lib/storage_service.dart`: secure profile and auth-session persistence
- `lib/terminal_client.dart`: websocket transport for share-session relay access

## What is still missing

There are still a few polish items left for the public product:

- OS-level external deep-link handling for opening `termviewer://...` links outside the app
- more UI polish for remote device states and session lifecycle messaging
- broader platform validation beyond the current Android debug build

## QR code rule

The mobile app must never expect QR codes containing machine credentials.

Correct behavior:

- machine credentials are used only to configure the PC agent
- QR codes contain a short-lived share-session token for the active sharing session

See `../docs/REMOTE_SESSION_FLOW.md` for the canonical public-server flow.

## Development

```bash
flutter pub get
flutter run
```

Useful files:

- `lib/main.dart`: app entry point and LAN flow
- `lib/public_access.dart`: remote-access UI and QR flow
- `lib/public_access_service.dart`: OIDC and remote API integration
- `lib/terminal_client.dart`: terminal transport and relay hooks
- `lib/storage_service.dart`: secure storage for saved hosts and public profiles
- `lib/terminal_themes.dart`: terminal theme support
