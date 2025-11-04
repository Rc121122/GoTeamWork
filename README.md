# GoTeamWork
Collaborative clipboard and chat application with real-time synchronization

## 📝 Description
GoTeamWork is a collaborative clipboard and chat application that allows seamless data sharing across devices. It supports two operational modes:

- **Host Mode**: Acts as a central server providing REST API for user/room management
- **Client Mode**: Connects to the host server with username authentication and waiting lobby

## 🚀 Features
- 📋 Real-time clipboard synchronization (Host mode)
- 💬 Group chat functionality (Host mode)
- 🖥️ Cross-platform GUI using Wails
- 🌐 REST API for central server communication
- 👥 User management with unique username validation
- 🏠 Room-based collaboration with automatic lifecycle management
- 🔄 Real-time user list updates (Client mode)
- 🎯 Invitation system for room joining

## 📦 Prerequisites

### 1. Go Installation
- Go 1.25.2 or higher
- Install from: https://go.dev/dl/

### 2. Wails Installation
Wails requires certain dependencies based on your operating system:

#### macOS
```bash
# Install Xcode Command Line Tools (if not already installed)
xcode-select --install

# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev

# Fedora
sudo dnf install gcc-c++ pkg-config gtk3-devel webkit2gtk3-devel

# Arch
sudo pacman -S gcc pkg-config gtk3 webkit2gtk

# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Windows
```bash
# Install Wails CLI (run in PowerShell or CMD)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Note: Requires WebView2 runtime (usually pre-installed on Windows 10/11)
# Download if needed: https://developer.microsoft.com/microsoft-edge/webview2/
```

## 🛠️ Setup Instructions

### 1. Clone the Repository
```bash
git clone https://github.com/Rc121122/GoTeamWork.git
cd GoTeamWork
```

### 2. Install Go Dependencies
```bash
# Download all required Go modules
go mod download

# Verify installation
go mod verify
```

### 3. Install Frontend Dependencies
```bash
# Install Node.js dependencies for the frontend
cd frontend
npm install
cd ..
```

### 4. Build the Application
```bash
# Check Wails installation and system requirements
wails doctor

# Build the application with frontend
wails build
```

## 🏃 Running the Application

### Host Mode (Central Server)
Run the application as a central server that provides REST API and manages users/rooms:

```bash
# Production build
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode host

# Development mode (with hot reload)
wails dev --mode host
```

**Host Mode Features:**
- Starts HTTP server on port 8080
- Provides REST API endpoints for client communication
- Shows chat/clipboard interface for the host user
- Manages user authentication and room creation

### Client Mode (User Interface)
Run the application as a client that connects to the host server:

```bash
# Production build
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode client

# Development mode (with hot reload)
wails dev --mode client
```

**Client Mode Features:**
- Username input screen with validation
- Waiting lobby showing all online users
- Real-time user list updates (every 5 seconds)
- Invite buttons for joining rooms

### Development Mode
```bash
# Run with Wails development server (hot reload)
wails dev

# The mode can be specified via command line or UI
```

### Production Build
```bash
# Build for your current platform
wails build

# Build for specific platforms
wails build -platform darwin/amd64    # macOS Intel
wails build -platform darwin/arm64    # macOS Apple Silicon
wails build -platform windows/amd64   # Windows
wails build -platform linux/amd64     # Linux
```

## 📚 Project Structure
```
GoTeamWork/
├── main.go              # Application entry point and mode selection
├── app.go               # Core application logic, user/room management, HTTP API
├── network.go           # Network operations (LAN scan, sync, connections)
├── go.mod               # Go module dependencies
├── go.sum               # Go module checksums
├── wails.json           # Wails configuration
├── frontend/            # Frontend application
│   ├── index.html       # Main HTML page
│   ├── package.json     # Node.js dependencies
│   ├── src/             # Frontend source code
│   │   ├── main.js      # Main JavaScript application
│   │   ├── style.css    # Global styles
│   │   └── app.css      # Component styles
│   └── wailsjs/         # Generated Wails bindings
├── build/               # Build output directory
│   └── bin/             # Compiled binaries
├── ApiMethod.md         # API methods documentation
├── design.md            # Design documentation
└── README.md            # This file
```

## 🔧 Configuration
Configuration options will be available in a config file (to be implemented):
- Network settings (ports, timeouts)
- UI preferences
- Server address for Central Mode
- Auto-start preferences

## 🐛 Troubleshooting

### Wails Build Issues
```bash
# Check system requirements
wails doctor

# Clean build cache
go clean -cache
rm -rf build/
```

### Mode Selection Issues
```bash
# Always use the built binary, not the go build output
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode host
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode client

# Don't use: ./GOproject --mode host (missing Wails build tags)
```

### Network Issues
- Ensure firewall allows the application (port 8080 for host mode)
- Check network permissions on macOS/Linux
- Host and clients must be able to communicate via HTTP

### Frontend Issues
```bash
# Rebuild frontend dependencies
cd frontend
rm -rf node_modules package-lock.json
npm install
cd ..
wails build
```

### API Connection Issues
- Verify host is running: `curl http://localhost:8080/api/users`
- Check that client can reach the host server
- Ensure CORS headers are properly configured

## 📖 Documentation
- [API Methods Documentation](ApiMethod.md) - Complete API reference
- [Wails Documentation](https://wails.io/docs/introduction)
- [Go Documentation](https://go.dev/doc/)

## 👨‍💻 Development
The application is fully implemented with the following components:

- **Backend**: Go application with Wails framework
- **Frontend**: HTML/CSS/JavaScript with modern UI
- **API**: REST endpoints for client-server communication
- **Modes**: Host mode (server) and Client mode (user interface)

### API Documentation
See [ApiMethod.md](ApiMethod.md) for detailed API method documentation.

### Architecture
- **Host Mode**: Provides central server functionality with REST API
- **Client Mode**: User interface for joining and collaborating
- **Room System**: Dynamic room creation and management
- **User Management**: Authentication and online status tracking

## 🔌 API Endpoints (Host Mode)

When running in host mode, the application provides REST API endpoints on `http://localhost:8080`:

### User Management
- `GET /api/users` - List all users
- `POST /api/users` - Create new user
- `GET /api/users/{id}` - Get specific user

### Room Management
- `GET /api/rooms` - List all rooms
- `POST /api/invite` - Invite user to room

### Usage Example
```bash
# Start host server
./build/bin/GOproject.app/Contents/MacOS/GOproject --mode host

# In another terminal, create a user
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"Alice"}' http://localhost:8080/api/users

# List all users
curl http://localhost:8080/api/users
```

## 📄 License
[Specify your license here]

## 🤝 Contributing
Contributions are welcome! Please feel free to submit a Pull Request.