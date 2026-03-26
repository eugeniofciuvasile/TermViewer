# Getting Started with TermViewer

TermViewer consists of two main parts: the **Agent** (running on your PC) and the **Mobile App** (running on your phone). For enterprise or remote access, there is also an optional **Public Server** stack.

## 1. Running the Agent (Go)

The Agent acts as a server that your phone connects to.

### Prerequisites
- [Go (Golang)](https://golang.org/dl/) 1.25.8+ installed on your PC.
- [Flutter/Dart SDK](https://flutter.dev/) ^3.11.3 for building the mobile app.
- (Optional) [Tmux](https://github.com/tmux/tmux) for shared session support.
- (Optional) [nFPM](https://nfpm.goreleaser.com/) for building `.deb` and `.rpm` packages.

### Building and Packaging
You can use the provided `Makefile` for easy management:

```bash
make build              # Build Go agent binary to dist/
make package            # Create .deb and .rpm via nFPM
make clean              # Remove build artifacts
```

### Running the Agent
1.  Run the built agent:
    ```bash
    ./dist/termviewer-agent --password mysecret
    ```

- This will generate a self-signed TLS certificate if it's the first run.
- The agent will start broadcasting its presence via mDNS on your local network.

### Command Line Options
- `--password`: Set the authentication password. Can also be set via `TERMVIEWER_PASSWORD` env var.
- `--port`: Set a custom port (default: 24242).
- `--command`: Run a custom shell command on startup (e.g., `bash`, `zsh`, `tmux attach`).
- `--attach <session_name>`: Attach to a native TermViewer session locally.
- `--relay-url`: Enterprise relay server WSS URL. Can also be set via `TERMVIEWER_RELAY_URL` env var.
- `--client-id`: Enterprise client ID. Can also be set via `TERMVIEWER_CLIENT_ID` env var.
- `--client-secret`: Enterprise client secret. Can also be set via `TERMVIEWER_CLIENT_SECRET` env var.
- `--tls-skip-verify`: Skip TLS verification for enterprise relay. Can also be set via `TERMVIEWER_TLS_SKIP_VERIFY` env var.

## 2. Using the Mobile App (Flutter)

### Running on your phone
1.  Ensure your phone and PC are on the same Wi-Fi network.
2.  Open the TermViewer app on your mobile device.
3.  The app will automatically discover any running TermViewer Agents on the LAN via **mDNS**.
4.  Tap on your PC's name in the "Discovered Agents" list.
5.  On first connection, the app uses **TOFU (Trust On First Use) certificate pinning** to verify the agent's identity.
6.  If you haven't entered the password yet, a modal will prompt you.
7.  Once connected, choose a session to attach to or start a new one.

### App Features
- **mDNS Auto-Discovery:** Automatically finds TermViewer agents on the local network.
- **TOFU Certificate Pinning:** Trust On First Use — the app remembers the agent's certificate after the first connection.
- **Virtual Canvas:** A 1200px-wide virtual canvas with pan and zoom gestures for comfortable terminal viewing.
- **Auto-Scrolling Viewport:** The view automatically follows terminal output while you are at the bottom.
- **Reset View:** Use the "Reset View" button (fullscreen_exit icon) to quickly reset zoom and pan.
- **Terminal Themes:** Choose from Dracula, Nord, Solarized Dark, and Monokai.
- **Custom Macros & Keybar Toolbar:** Define custom macros and access common keys from a configurable toolbar.
- **Bidirectional File Transfer:** Browse and transfer files between your phone and PC with a built-in file browser UI.
- **Bidirectional Clipboard Sync:** Copy and paste seamlessly between your phone and the remote terminal.
- **System HUD:** Monitor CPU, RAM, disk usage, and uptime at a glance.
- **Terminal Recording:** Record terminal sessions in Asciinema v2 `.cast` format.
- **Session Types:** Supports both native TermViewer sessions and Tmux sessions.
- **Session Multiplexing:** Multiple clients can connect to the same PTY simultaneously.

## 3. Advanced Usage: CLI Attachment

You can also use the TermViewer agent itself to attach to sessions running in the background:

```bash
# Start a session in the background
./dist/termviewer-agent --password mysecret

# Attach to it from another terminal
./dist/termviewer-agent --attach main
```

When you type `exit` in the attached shell, the session will be closed on both the host and any connected mobile apps.

## 4. Public Server (Development Setup)

For remote access beyond your LAN, you can run the public server stack locally for development.

### Infrastructure
```bash
cd server/infrastructure
cp .env.template .env
docker compose up -d
```

### Backend
```bash
cd server/backend
cp .env.template .env
go run .
```

> The backend supports `DATABASE_URL` or individual `DB_*` variables for database configuration.

### Frontend
```bash
cd server/frontend
cp .env.template .env.local
npm install && npm run dev
```

> **Note:** Generate a strong `NEXTAUTH_SECRET` for your environment. The `.env.template` files in each directory are committed as starting points.

## 5. Production Deployment

```bash
cd deploy
./setup.sh              # Interactive setup (testing or production)
cd testing|production
docker compose up -d --build
```

## 6. Connecting the Agent to a Public Server

To connect an agent to your deployed public server:

```bash
./dist/termviewer-agent \
  --relay-url wss://yourdomain.com/ws/relay/agent \
  --client-id YOUR_CLIENT_ID \
  --client-secret YOUR_CLIENT_SECRET
```

These values can also be set via environment variables (`TERMVIEWER_RELAY_URL`, `TERMVIEWER_CLIENT_ID`, `TERMVIEWER_CLIENT_SECRET`).
