# GoTeamWork
go final project

## 📝 Description
GoTeamWork is a collaborative clipboard and chat application that allows seamless data sharing across devices. It supports two modes:
- **LAN Mode**: Direct peer-to-peer communication on local networks
- **Central Mode**: Server-based synchronization for remote devices

## 🚀 Features
- 📋 Real-time clipboard synchronization
- 💬 Group chat functionality
- 🔄 Automatic peer discovery (LAN mode)
- 🖥️ Cross-platform GUI using Wails
- 🔒 Secure data transmission

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

### 3. Initialize Wails Project (if building UI)
```bash
# Check Wails installation
wails doctor

# Build the application
wails build

# Or run in development mode with hot reload
wails dev
```

## 🏃 Running the Application

### Development Mode
```bash
# Run with Wails development server (hot reload)
wails dev
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

### Run Directly with Go
```bash
# Run without UI (terminal mode)
go run .
```

## 📚 Project Structure
```
GoTeamWork/
├── main.go         # Application entry point and mode selection
├── network.go      # Network operations (LAN scan, sync, connections)
├── UI.go           # User interface components and clipboard/chat handlers
├── go.mod          # Go module dependencies
└── README.md       # This file
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

### Network Issues
- Ensure firewall allows the application
- Check network permissions on macOS/Linux
- Verify peers are on the same subnet (LAN mode)

## 📖 Documentation
- [Wails Documentation](https://wails.io/docs/introduction)
- [Go Documentation](https://go.dev/doc/)

## 👨‍💻 Development
Currently in design phase. Implementation coming soon.

## 📄 License
[Specify your license here]

## 🤝 Contributing
Contributions are welcome! Please feel free to submit a Pull Request.