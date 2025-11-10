## Objectives:
Tree Structure, 3 main modules:
- main.go: Entry point
    - mode selection (LAN/ Central-server client/ Central-server server)
    - get&check os permissions
    - hotkey assignment
    - data sync scheduling(LAN mode)

- apps.go: UI controller
    - clipboard management
    - chat interface
    - room status (attendants, connection status)

- network.go: Network operations
    - Central server communication
    - LAN discovery

## Members:
- mr.Lin
- mr.Chang
- mr.Ho

### Weekly goal:
- Week 1 (11/4):
  1. Central-server implement(join/leave room, chatroom) - Assigned to mr.Lin
     - ~~Implement join room functionality in app.go~~ ✅
     - ~~Implement leave room functionality in app.go~~ ✅ (Backend LeaveRoom() exists, needs API endpoint)
     - ~~Implement chat message sending and receiving in app.go~~ ✅
  2. UI(chatroom, connection status) - Assigned to mr.Chang
     - ~~Build chatroom interface in frontend~~ ✅
     - ~~Add connection status display in UI~~ ✅
  3. network.go(connect to central server) - Assigned to mr.Ho
     - ~~Add connection logic to central server in network.go~~ ✅
     - ~~Handle connection establishment and errors~~ ✅
     - ~~Fetch data to local db~~ ✅

- Week 2 (11/11):
  1. Password for rooms(Central-server) - Assigned to mr.Lin
     - Implement password authentication for joining rooms on central server
     - Add password validation and secure storage
  2. Clipboard permission on OS + hotkey assignment - Assigned to mr.Chang
     - Check and request clipboard permissions across OS (Windows, macOS, Linux)
     - Implement hotkey assignment for clipboard operations
  3. UI clipboard thumbnail - Assigned to mr.Ho
     - Create UI component to display clipboard content thumbnails
     - Integrate thumbnail display with clipboard management

- Week 3 (11/18):
  1. LAN search & invite - Assigned to mr.Lin
     - Implement LAN device discovery mechanism
     - Add invite functionality for connecting to LAN peers
  2. LAN successor selection - Assigned to mr.Chang
     - Implement logic for selecting successor in LAN mode
     - Handle failover and leader election scenarios
  3. LAN data sync - Assigned to mr.Ho
     - Implement data synchronization across LAN peers
     - Ensure data consistency and conflict resolution

- Week 4 (11/25):
  1. Mode selection implementation - Assigned to mr.Lin
     - Implement mode selection (LAN/Central-server client/server) in main.go
     - Add UI for mode selection if needed
  2. OS permissions check - Assigned to mr.Chang
     - Implement get and check OS permissions in main.go
     - Handle permission requests and errors
  3. Hotkey assignment - Assigned to mr.Ho
     - Implement hotkey assignment functionality
     - Integrate with clipboard and other operations

- Week 5 (12/2):
  1. Integration testing - Assigned to mr.Lin
     - Perform end-to-end integration tests for all modules
     - Test cross-module interactions
  2. Bug fixes and optimization - Assigned to mr.Chang
     - Identify and fix bugs from testing
     - Optimize performance and code quality
  3. Documentation - Assigned to mr.Ho
     - Write user documentation and API docs
     - Update README and design documents

- Week 6 (12/9): Complete whole project - Assigned to all members
  - Final integration and testing
  - Code review and final adjustments
  - Prepare for deployment and presentation

- Week 7 (12/16): Presentation - Assigned to all members
  - Prepare presentation slides and demo
  - Rehearse and finalize project deliverables
  hello