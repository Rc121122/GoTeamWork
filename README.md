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

## Prerequisites

- Go 1.25.2 or higher
- Node.js and npm (for frontend)
- Wails v2 CLI

### Installing Dependencies

#### macOS
```bash
xcode-select --install
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev

go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Windows
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
# Ensure WebView2 runtime is installed
```

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/GoTeamWork.git
cd GoTeamWork
```

2. Install Go dependencies:
```bash
go mod download
```

3. Install frontend dependencies:
```bash
cd frontend
npm install
cd ..
```

4. Build the application:
```bash
wails build
```

## Usage

### Running as Host (Server)
```bash
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode host
```

The host provides:
- REST API server on port 8080
- WebSocket connections for real-time updates
- Clipboard sharing interface
- Chat functionality

### Running as Client
```bash
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode client
```

Clients can:
- Connect to host servers
- Join rooms via invitations
- Share clipboard content
- Participate in group chat

### Development Mode
```bash
wails dev --mode host
# or
wails dev --mode client
```

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
├── auth_test.go            # Authentication tests
├── clipboard_test.go       # Clipboard functionality tests
├── clip.go                 # Clipboard operations
├── clip_hotkey_test.go     # Hotkey tests
├── handlers_test.go        # API handler tests
├── history_hash_test.go    # History hash tests
├── history_test.go         # History tests
├── sanitize_test.go        # Sanitization tests
├── sse_test.go             # SSE tests
├── sse_routes_test.go      # SSE routes tests
├── go.mod                  # Go module file
├── wails.json              # Wails configuration
├── frontend/               # Frontend application
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.tsx
│       ├── main.tsx
│       ├── state.ts
│       ├── client.ts
│       ├── host.ts
│       ├── sse.ts
│       ├── api/
│       │   ├── httpClient.ts
│       │   └── types.ts
│       └── components/
│           ├── HostDashboard.tsx
│           ├── HUD.tsx
│           ├── LandingPage.tsx
│           ├── Lobby.tsx
│           ├── Modals.tsx
│           ├── NewUserPage.tsx
│           ├── Room.tsx
│           ├── Sidebar.tsx
│           └── TitleBar.tsx
├── build/                  # Build output
├── docs/                   # Documentation
├── internal/               # Internal packages
├── test_clip/              # Test utilities
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

### Building for Different Platforms
```bash
# macOS Intel
wails build -platform darwin/amd64

# macOS Apple Silicon
wails build -platform darwin/arm64

# Windows
wails build -platform windows/amd64

# Linux
wails build -platform linux/amd64
```

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

## License

[Add your license information here]

## Support

For issues and questions:
- Check the [API documentation](docs/ApiMethod.md)
- Review [design documentation](docs/design.md)
- Open an issue on GitHub