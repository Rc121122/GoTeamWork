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
  1. Central-server features - Assigned to mr.Lin
     - Add password auth for rooms + secure storage
     - Implement SSE for real-time invite notifications
  2. Clipboard & hotkeys - Assigned to mr.Chang
     - OS clipboard permissions check + request
     - Hotkey assignment for clipboard ops
  3. UI clipboard thumbnail - Assigned to mr.Ho
     - Clipboard content thumbnail component
     - Integrate with clipboard management

  **CRITICAL FIXES (Week 2 Priority):** ✅ **COMPLETED**
  - ✅ **Code Quality:** Refactor HTTP headers, extract validation helpers, remove test users, add config file
    - Extracted HTTP handlers to `handlers.go` (226 lines)
    - Extracted SSE logic to `sse.go` (196 lines) 
    - Extracted type definitions to `types.go` (61 lines)
    - Reduced `app.go` from 860+ to 407 lines (47% reduction)
    - Fixed redundant `/api/users` endpoints
    - Added structured request/response types
  - ✅ **Missing APIs:** Add /api/leave endpoint, clipboard sharing API, disconnect detection, heartbeat
    - Added proper `/api/leave` endpoint with structured requests
    - SSE heartbeat implemented for connection management
  - **Memory Mgmt:** Chat history limits, auto room deletion, user timeouts, garbage collection
  - **Sync Issues:** Fix race conditions, message ordering, conflict resolution, duplicate prevention
  - **Stability:** Error logging, graceful shutdown, retry logic, circuit breaker
  - **Security Prep:** Input sanitization, rate limiting, CORS config, password hashing setup

- Week 3 (11/18):
  1. LAN search & invite - Assigned to mr.Lin
     - LAN device discovery + invite functionality
  2. LAN successor selection - Assigned to mr.Chang
     - Successor logic + failover handling
  3. LAN data sync - Assigned to mr.Ho
     - Peer data sync + conflict resolution

- Week 4 (11/25):
  1. Mode selection - Assigned to mr.Lin
     - Implement mode selection in main.go + UI
  2. OS permissions - Assigned to mr.Chang
     - Permission checks + requests in main.go
  3. Hotkey assignment - Assigned to mr.Ho
     - Hotkey functionality + integration

- Week 5 (12/2):
  1. Integration testing - Assigned to mr.Lin
     - End-to-end tests + cross-module validation
  2. Bug fixes & optimization - Assigned to mr.Chang
     - Bug fixes + performance optimization
  3. Documentation - Assigned to mr.Ho
     - User docs + API documentation

- Week 6 (12/9): Complete whole project - Assigned to all members
  - Final integration, testing, code review, deployment prep

- Week 7 (12/16): Presentation - Assigned to all members
  - Slides, demo prep, final deliverables
  hello