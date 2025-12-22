# Release v1.0.0

## Overview

GoTeamWork v1.0.0 is the first stable release of our real-time collaborative clipboard sharing and chat application. Built with Go and Wails, this cross-platform desktop app enables seamless team collaboration by synchronizing clipboard content and facilitating instant messaging across devices.

## What's New

### Major Features
- **Host/Client Architecture**: Run as a central host server or connect as a client to join existing sessions.
- **Real-time Clipboard Synchronization**: Automatically share text, images, and files between connected devices.
- **Instant Chat**: Room-based messaging with Server-Sent Events (SSE) for real-time communication.
- **File Transfer**: Drag-and-drop file sharing with compression support.
- **Network Discovery**: Automatic LAN discovery using Zeroconf (Bonjour).
- **Transparent HUD**: Floating gopher icon for clipboard sharing notifications.
- **Cross-Platform Support**: Native builds for macOS and Windows.

### Technical Improvements
- Mode Selection UI: New interface to choose between host and client modes at startup.
- Enhanced Error Handling: Idiomatic Go error handling throughout the codebase.
- Concurrency Optimizations: Extensive use of Goroutines and Channels for performance.
- Comprehensive Testing: Unit tests, benchmarks, and CI/CD integration.

### Bug Fixes
- Resolved Windows client mode crashes related to clipboard monitoring.
- Fixed HUD positioning and always-on-top behavior.
- Improved network client retry logic with exponential backoff.

## Installation

### Prerequisites
- **macOS**: 10.13 or higher
- **Windows**: 10 or higher
- No additional dependencies required - the app is self-contained.

### Download
Download the appropriate installer for your platform from the assets below:

- **macOS**: `GoTeamWork.dmg` (Built on macOS 15.3)
- **Windows**: `GoTeamWork.exe` (Built on Windows 11)

### Installation Steps

#### macOS
1. Download `GoTeamWork.dmg`.
2. Double-click the DMG file to mount it.
3. Drag `GoTeamWork.app` to your Applications folder.
4. Eject the DMG and launch GoTeamWork from Applications.
5. On first launch, you may need to right-click the app → Open (or System Settings → Privacy & Security → Open Anyway). If quarantined, clear with: `xattr -cr /Applications/GoTeamWork.app`.

#### Windows
1. Download `GoTeamWork.exe`.
2. Run the installer executable.
3. Follow the on-screen instructions to complete installation.
4. Launch GoTeamWork from the Start menu or desktop shortcut.

### First Run
- The app will prompt you to select **Host** or **Client** mode.
- **Host**: Starts the server for others to connect.
- **Client**: Connects to an existing host by entering the server address.

## Usage

### Basic Operation
1. Choose your mode (Host/Client).
2. If Client, enter the host's IP address and port (default: 8080).
3. Create or join a room.
4. Start sharing clipboard content and chatting.

### Hotkeys
- **Copy Hotkey**: Automatically detects ctrl+c/cmd+c to open gopher HUD
- **Share Clipboard**:  Click the floating gopher icon to share clipboard when copying
- **HUD Toggle**: Click setting on the navigation bar (side bar in main window) to show/hide the HUD.

### Advanced Features
- **LAN Discovery**: Hosts are automatically discoverable on the local network. (Port 8080)
- **File Compression**: Large files are compressed before transfer.
- **User Management**: Unique usernames with validation.

## Known Issues
- Windows: Occasional clipboard monitoring delays in high-load scenarios.
- macOS: Gatekeeper may block first launch - follow bypass instructions above.
- Network: Firewall settings may need adjustment for LAN discovery.

## Contributing
This is an open-source project. Contributions are welcome! See the repository for development setup and contribution guidelines.

## Support
- **Issues**: Report bugs on GitHub Issues.
- **Discussions**: Join community discussions for questions and feedback.
- **Documentation**: Full docs available in the `docs/` directory.


Thank you for using GoTeamWork! 🎉