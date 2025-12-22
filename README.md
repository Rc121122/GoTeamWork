# GoTeamWork

A real-time collaborative clipboard sharing and chat application built with Go and Wails, enabling seamless data synchronization across devices in team environments.

## Features

- **Real-time Clipboard Synchronization**: Share clipboard content instantly between host and connected clients
- **Group Chat**: Built-in chat functionality for team communication
- **Cross-Platform GUI**: Native desktop application using Wails framework
- **Host/Client Architecture**: Flexible deployment with central host server and multiple clients
- **User Management**: Username-based authentication with unique validation
- **Room System**: Dynamic room creation and invitation-based joining
- **REST API**: Comprehensive API for client-server communication
- **LAN Discovery**: Automatic network scanning for host discovery
- **Hotkey Integration**: Clipboard sharing triggered by customizable hotkeys
- **Transparent HUD**: Floating gopher icon for clipboard sharing notifications

## Prerequisites

- **Go**: 1.25.2 or higher
- **Node.js**: 18.x or higher (with npm)
- **Wails CLI**: v2.10.2 or higher
- **Git**: For cloning the repository

### Platform-Specific Requirements

#### macOS
- Xcode Command Line Tools
- macOS 10.13 or higher

#### Windows
- Windows 10 or higher
- WebView2 Runtime (automatically installed with Wails, or download from [Microsoft](https://developer.microsoft.com/microsoft-edge/webview2/))
- GCC (MinGW-w64 recommended) for CGO compilation

#### Linux
- GTK3 development libraries
- WebKitGTK development libraries

## Installation

### 1. Install Wails CLI (v2.10.2)

#### macOS
```bash
# Install Xcode Command Line Tools if not already installed
xcode-select --install

# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
```

#### Windows
```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2

# Ensure WebView2 Runtime is installed (Wails will prompt if needed)
# Download from: https://developer.microsoft.com/microsoft-edge/webview2/
```

#### Linux (Ubuntu/Debian)
```bash
# Install system dependencies
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev

# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
```

### 2. Clone and Setup Project
```bash
# Clone the repository
git clone https://github.com/Rc121122/GoTeamWork.git
cd GoTeamWork

# Install Go dependencies
go mod download

# Install frontend dependencies
cd frontend
npm install
cd ..
```

### 3. Build the Application

#### Build for Current Platform
```bash
wails build
```

#### Build for Specific Platforms

**macOS Intel:**
```bash
wails build -platform darwin/amd64
```

**macOS Apple Silicon:**
```bash
wails build -platform darwin/arm64
```

**Windows (64-bit):**
```bash
wails build -platform windows/amd64
```

**Linux (64-bit):**
```bash
wails build -platform linux/amd64
```

The built application will be in `build/bin/` directory.

## Usage

### Running the Application

By default (no flags), the app starts and prompts you to choose **Host** or **Client** in the UI. You can still force a mode via the `--mode` flag if you prefer CLI control.

#### macOS
```bash
# Host mode
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode host

# Client mode
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode client

# Prompt for mode (default)
./build/bin/GOproject.app/Contents/MacOS/GOproject
```

> Note: The app is not code-signed. On first launch, you may need to bypass Gatekeeper by right-clicking the app → Open (or System Settings → Privacy & Security → Open Anyway). If it was quarantined, clear the flag with `xattr -cr build/bin/GOproject.app`.

#### Windows
```bash
# Host mode
.\build\bin\GOproject.exe --mode host

# Client mode
.\build\bin\GOproject.exe --mode client

# Prompt for mode (default)
.\build\bin\GOproject.exe
```

#### Linux
```bash
# Host mode
./build/bin/GOproject --mode host

# Client mode
./build/bin/GOproject --mode client

# Prompt for mode (default)
./build/bin/GOproject
```

### Development Mode
```bash
# Host mode
wails dev --mode host

# Client mode
wails dev --mode client
```

### Application Modes

**Host Mode:**
- Runs the central server
- Provides REST API on port 8080
- Manages WebSocket connections
- Handles clipboard sharing and chat

**Client Mode:**
- Connects to host servers
- Discovers hosts via LAN scanning
- Joins rooms via invitations
- Shares clipboard and participates in chat

## Project Structure

```
GoTeamWork/
├── main.go                 # Application entry point
├── app.go                  # Core application logic
├── network.go              # Network utilities and LAN scanning
├── types.go                # Data structures and types
├── handlers.go             # HTTP API handlers
├── sse.go                  # Server-Sent Events implementation
├── sanitize.go             # Input sanitization utilities
├── clip.go                 # Clipboard operations
├── clip_helper/            # Platform-specific clipboard helpers
├── auth_test.go            # Authentication tests
├── clipboard_test.go       # Clipboard functionality tests
├── handlers_test.go        # API handler tests
├── go.mod                  # Go module file
├── wails.json              # Wails configuration
├── frontend/               # React frontend
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.tsx
│       ├── main.tsx
│       ├── components/
│       │   ├── HUD.tsx          # Transparent floating HUD
│       │   ├── HostDashboard.tsx
│       │   └── ...
│       └── api/
├── build/                  # Build output
├── docs/                   # Documentation
├── internal/               # Internal packages
├── test_clip/              # Clipboard test utilities
└── tests/                  # Test files
```

## API Documentation

### User Management
- `POST /api/users` - Create user
- `GET /api/users` - List users
- `GET /api/users/{id}` - Get user details

### Room Management
- `GET /api/rooms` - List rooms
- `POST /api/invite` - Send room invitation
- `POST /api/invite/accept` - Accept invitation
- `POST /api/join` - Join room

### Authentication
- JWT-based authentication required for API access
- Set `JWT_SECRET` environment variable for production

## Configuration

### Environment Variables
- `JWT_SECRET`: Secret key for JWT token generation (required for production)

### Build Configuration
Modify `wails.json` for build settings and platform targets.

## Development

### Running Tests
```bash
go test ./...
```

### Code Signing (macOS)
```bash
codesign --force --deep --sign - GOproject.app
#or
xattr -crw GOproject.app
```

### Troubleshooting

#### Common Issues

**Build fails on Windows:**
- Ensure MinGW-w64 is installed and in PATH
- Install WebView2 Runtime
- Run as Administrator if permission issues occur

**Transparent HUD not working:**
- Ensure `WindowIsTranslucent: true` in `main.go` for Windows builds
- Check that frontend CSS uses `background: transparent`

**Network connectivity issues:**
- Ensure firewall allows connections on port 8080
- Check LAN discovery settings

**Clipboard sharing not working:**
- On macOS: Grant accessibility permissions
- On Windows: Ensure proper clipboard permissions

## Security

- Input sanitization implemented
- JWT authentication for API access
- No hardcoded secrets in production builds
- Environment-based configuration

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Check the [API documentation](docs/ApiMethod.md)
- Review [design documentation](docs/design.md)
- Open an issue on GitHub
