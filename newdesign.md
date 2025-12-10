# New Design Document for GoTeamWork

## 1. Incomplete Goals

Based on the current project state (as of 2025年12月9日), the following goals from the original [`design.md`](design.md ) remain incomplete. These are prioritized by dependency and impact, focusing on LAN features, stability, and finalization. Completed items (e.g., central-server join/leave/chat, UI chatroom, SSE invites, clipboard permissions/hotkeys, code refactoring, and missing APIs) are excluded.

### High Priority (Core Functionality Gaps)
- **Week 3 (11/18) - LAN Features**:
  - LAN device discovery + invite functionality (Assigned to mr.Lin): Implement peer discovery using the existing [`lan_scanner.go`](lan_scanner.go ) logic, integrated into [`network.go`](network.go ) for LAN mode. Add invite mechanisms for peer-to-peer connections.
  - LAN successor selection + failover handling (Assigned to mr.Chang): Design logic for electing a "successor" peer in case of disconnection (e.g., via heartbeat and leader election). Implement failover to maintain data sync.
  - LAN data sync + conflict resolution (Assigned to mr.Ho): Add peer-to-peer data synchronization (clipboard/chat) with conflict resolution (e.g., last-write-wins or operational transformation). Integrate with [`app.go`](app.go ) history pool.

- **Week 4 (11/25) - Mode Selection & Permissions**:
  - Implement mode selection in [`main.go`](main.go ) + UI (Assigned to mr.Lin): Extend beyond command-line flags to include a UI-based selector (see Section 2 for details).
  - Permission checks + requests in [`main.go`](main.go ) (Assigned to mr.Chang): Generalize clipboard permissions to broader OS checks (e.g., network access for LAN scanning).
  - Hotkey functionality + integration (Assigned to mr.Ho): Expand clipboard hotkeys to include custom assignable hotkeys for modes (e.g., toggle LAN sync).

- **Week 2 (11/11) - Critical Fixes (Remaining)**:
  - Memory Mgmt: Implement auto room deletion, user timeouts, and garbage collection (e.g., periodic cleanup in [`app.go`](app.go )).
  - Sync Issues: Add race condition fixes, message ordering, and duplicate prevention (e.g., sequence IDs in operations).
  - Stability: Enhance error logging, graceful shutdown, and circuit breaker (e.g., in [`network.go`](network.go )).
  - Security Prep: Add rate limiting, CORS config, and password hashing (e.g., for user auth in central-server mode).

### Medium Priority (Testing & Polish)
- **Week 5 (12/2)**:
  - Integration testing: Develop end-to-end tests for cross-module validation (e.g., full workflows for central-server and LAN modes).
  - Bug fixes & optimization: Address any runtime issues and optimize performance (e.g., reduce latency in clipboard sync).
  - Documentation: Complete user docs and API documentation (expand [`README.md`](README.md ), [`ApiMethod.md`](ApiMethod.md ), and [`SSEroute.md`](SSEroute.md )).

- **Week 6 (12/9) - Project Completion**:
  - Final integration, testing, code review, and deployment prep: Ensure all modes work seamlessly, add CI/CD, and prepare for release.

- **Week 7 (12/16) - Presentation**:
  - Slides, demo prep, and final deliverables: Create presentation materials showcasing all modes.

### Additional Notes
- **Dependencies**: LAN features depend on mode selection. Stability fixes should be tackled early to avoid regressions.
- **Timeline Adjustment**: With the current date at 12/9, shift focus to Weeks 3-4 for LAN/mode integration, then 5-6 for testing/completion.
- **Risks**: LAN mode introduces complexity (e.g., NAT traversal for peer discovery); consider libraries like `golang.org/x/net/ipv4` for UDP broadcasting if needed.

## 2. Mode Selection Tree and Integration Design

### Mode Selection Tree
The application supports three primary modes, structured as follows for clarity and extensibility:

```
Mode Selection
├── Central Server Mode (Client-server architecture for scalable, centralized collaboration)
│   ├── Host Mode (Acts as central server: manages rooms, users, clipboard sync, and chat via REST API + SSE)
│   └── Client Mode (Connects to host: joins rooms, syncs data, and participates in chat)
└── LAN Mode (Peer-to-peer architecture for local, ad-hoc collaboration without internet dependency)
    └── Peer Mode (All instances are equal peers: discover neighbors, sync data directly, handle invites/failover)
```

- **Central Server Mode**: Ideal for remote teams; requires a designated host (e.g., on a public server). Clients connect via HTTP/WebSocket.
- **LAN Mode**: For local networks; no central server needed, but limited to subnet scope. Peers auto-discover and sync.
- **Hybrid Potential**: Future extension could allow switching between modes mid-session (e.g., start in LAN, escalate to central-server if needed).

### Design for Combining Modes in a Single Executable
To unify all modes into one executable (avoiding separate builds like the current [`lan_scanner.go`](lan_scanner.go ) tag), implement the following architecture. This ensures a single binary that dynamically adapts based on user selection, while keeping code modular.

#### Key Principles
- **Single Entry Point**: Use [`main.go`](main.go ) as the unified launcher.
- **Mode-Agnostic Core**: Shared components (e.g., clipboard management in [`app.go`](app.go ), UI in frontend) work across modes.
- **Conditional Logic**: Branch behavior based on selected mode (no build tags; use runtime flags/structs).
- **UI-First Selection**: Prioritize user-friendly mode selection over command-line only.

#### Implementation Steps
1. **Mode Selection Mechanism**:
   - **UI Dialog at Startup**: On launch, show a modal dialog (via Wails frontend) for mode selection if no flag is provided. Options: "Central Server Host", "Central Server Client", "LAN Peer". Persist last selection in a config file (e.g., [`config.json`](network.go )).
   - **Command-Line Fallback**: Retain/extend the [`--mode`](app.go ) flag: [`--mode host`](app.go ), [`--mode client`](app.go ), [`--mode lan`](app.go ). If set, skip UI and proceed directly.
   - **Validation**: Ensure prerequisites (e.g., network permissions for LAN, server URL for client mode).

2. **Unified App Initialization**:
   - In [`main.go`](main.go ), after mode selection, create an `App` instance with a [`Mode`](app.go ) enum (e.g., `ModeCentralHost`, `ModeCentralClient`, `ModeLAN`).
   - Pass mode to [`NewApp(mode)`](app.go ), which configures shared state (e.g., clipboard monitor) and mode-specific logic.

3. **Mode-Specific Modules**:
   - **Central Server Mode**:
     - Host: Enable server endpoints in [`handlers.go`](handlers.go )/[`sse.go`](sse.go ); start HTTP server.
     - Client: Use `NetworkClient` in [`network.go`](network.go ) to connect to host; disable local server.
   - **LAN Mode**:
     - Use central-server like hosting: The first user (who starts LAN mode) becomes the initial host, invites others to join as clients, and creates a queue of users based on join order.
     - If the host leaves, the next user in the queue automatically becomes the new host, ensuring seamless failover.
     - Integrate [`lan_scanner.go`](lan_scanner.go ) logic into [`network.go`](network.go ) (remove build tag; make it a function like `DiscoverLANPeers()` for initial discovery).
     - Host runs mini-server (reuse central server endpoints); clients connect via HTTP to host's IP/port.
     - Invites: Host broadcasts invites via UDP; clients accept and join the queue.
     - Failover: Queue-based succession (no complex election needed).
   - **Shared Network Layer**: Extend [`network.go`](network.go ) with a `NetworkMode` interface (e.g., `CentralServer` vs. `LANPeer`), allowing polymorphic behavior.

4. **Data Sync Scheduling**:
   - For LAN: Schedule periodic syncs (e.g., every 5s) using a goroutine in [`app.go`](app.go ), triggered only in LAN mode.
   - For Central: Rely on SSE for real-time push.

5. **UI/Frontend Integration**:
   - Add mode-specific UI elements (e.g., server URL input for client mode, peer list for LAN).
   - Use Wails bindings to expose mode status (e.g., `GetCurrentMode()`).

6. **Permissions & Hotkeys**:
   - In [`main.go`](main.go ), check OS permissions early (extend [`clip.go`](clip.go ) logic to include network access prompts).
   - Hotkeys: Make assignable via UI, with defaults per mode (e.g., LAN sync toggle).

7. **Build & Deployment**:
   - Single `go build` command; no tags needed.
   - Embed mode logic in the binary; use feature flags if size is a concern.

#### Benefits
- **Simplicity**: One executable reduces confusion and maintenance.
- **Flexibility**: Users can switch modes without rebuilding.
- **Scalability**: Central mode for large groups, LAN for quick local sessions.

#### Potential Challenges & Mitigations
- **Complexity**: LAN peer discovery may fail on complex networks (e.g., VPNs); add manual IP input as fallback.
- **Security**: LAN mode lacks central auth; add optional shared secrets.
- **Testing**: Add mode-specific unit tests; use Docker for simulated networks.

This design builds on the existing codebase while addressing gaps. Implement incrementally: Start with UI mode selection, then LAN integration. Let me know if you need code snippets or further details!