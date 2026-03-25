# Getting Started with TermViewer

TermViewer consists of two main parts: the **Agent** (running on your PC) and the **Mobile App** (running on your phone).

## 1. Running the Agent (Go)

The Agent acts as a server that your phone connects to.
### Prerequisites
- [Go (Golang)](https://golang.org/dl/) 1.18+ installed on your PC.
- (Optional) [Tmux](https://github.com/tmux/tmux) for shared session support.
- (Optional) [nFPM](https://nfpm.goreleaser.com/) for building packages.

### Building and Packaging
You can use the provided `Makefile` for easy management:

```bash
# Build the agent binary
make build

# Create .deb and .rpm packages (requires nFPM)
make package
```

### Running the Agent
1.  Run the built agent:
    ```bash
    ./dist/termviewer-agent --password mysecret
    ```

- This will generate a self-signed TLS certificate if it's the first run.
- The agent will start broadcasting its presence via mDNS on your local network.

### Command Line Options
- `--password`: Set the authentication password (required).
- `--port`: Set a custom port (default: 24242).
- `--command`: Run a custom shell command on startup (e.g., `bash`, `zsh`, `tmux attach`).
- `--attach <session_name>`: Attach to a native TermViewer session locally.

## 2. Using the Mobile App (Flutter)

### Running on your phone
1.  Ensure your phone and PC are on the same Wi-Fi network.
2.  Open the TermViewer app on your mobile device.
3.  The app will automatically discover any running TermViewer Agents on the LAN.
4.  Tap on your PC's name in the "Discovered Agents" list.
5.  If you haven't entered the password yet, a modal will prompt you.
6.  Once connected, choose a session to attach to or start a new one.

### App Features
- **Virtual Desktop Canvas:** Panning and zooming allowed for large terminal windows.
- **Auto-scrolling:** The view automatically follows the terminal output while you are at the bottom.
- **Reset View:** Use the "Reset View" button (fullscreen_exit icon) to quickly reset zoom and pan.
- **Session Refresh:** Manually refresh the session list or wait for it to auto-refresh when switching sessions.

## 3. Advanced Usage: CLI Attachment

You can also use the TermViewer agent itself to attach to sessions running in the background:

```bash
# Start a session in the background
./dist/termviewer-agent --password mysecret

# Attach to it from another terminal
./dist/termviewer-agent --attach main
```

When you type `exit` in the attached shell, the session will be closed on both the host and any connected mobile apps.
