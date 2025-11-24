# JavaScript & HTTP API Reference

The desktop client communicates with the Go backend through two layers:

- **Wails bindings** exposed under `frontend/wailsjs/go/main/App`. These are Promise-based helpers generated from exported Go methods on `App`.
- **HTTP + SSE endpoints** served by the host instance (`StartHTTPServer` binds to `http://localhost:8080`). The web client (`frontend/src`) uses these endpoints directly.

All TypeScript models referenced below are defined in `frontend/wailsjs/go/models.ts`.

## Wails Bindings (`main.App`)

### Application & Connectivity
- `GetMode(): Promise<string>` → Returns `'host'` or `'client'` based on launch flag.
- `GetConnectionStatus(): Promise<boolean>` → Client-only helper that reports the `NetworkClient` connectivity state.
- `StartHTTPServer(port: string): Promise<void>` → Host-only. Starts the embedded REST/SSE server (already invoked automatically on startup in host mode).
- `Greet(name: string): Promise<string>` → Simple diagnostics helper used in samples.

### User Management
- `ListAllUsers(): Promise<Array<main.User>>` → Host-side map snapshot of known users.
- `CreateUser(name: string): Promise<main.User>` → Host-only creator that emits a `user_created` SSE event.

### Room Management
- `GetAllRooms(): Promise<Array<main.Room>>` → Returns every room tracked by the host.
- `GetCurrentRoom(): Promise<main.Room | null>` → Host-side pointer to the room created via `Invite`. Returns `null` when no active room.
- `CreateRoom(name: string): Promise<main.Room>` → Host-only explicit room creation. Emits `room_created` SSE events.
- `Invite(userId: string): Promise<string>` → Host-only convenience that lazily creates the current room (if needed), adds the user, and emits `user_invited` SSE payloads.
- `LeaveRoom(userId: string): Promise<string>` → Removes a user from their room; auto-deletes rooms that fall below two members.

### Chat
- `SendChatMessage(roomId: string, userId: string, message: string): Promise<string>` → Saves the message via `ChatPool` and emits `chat_message` SSE events to other room members.
- `GetChatHistory(roomId: string): Promise<Array<main.ChatMessage>>` → Returns the stored chat transcript for the provided room.

## REST Endpoints (host mode)

Unless otherwise noted, responses are JSON. Request DTOs live in `types.go`.

- `GET /api/users` → `main.User[]` snapshot.
- `POST /api/users { name: string }` → Creates a user. Returns `201` with `main.User` or `409` if the name is already taken.
- `GET /api/users/{id}` → Retrieves a single user or returns `404`.
- `GET /api/rooms` → `main.Room[]` describing current rooms.
- `POST /api/rooms { name: string }` → Explicit room creation (host dashboards, tests).
- `POST /api/invite { userId: string, inviterId: string }` → Sanitizes the payload, ensures/creates the inviter's room, and emits an SSE invite for the target user so they can accept via `/api/join`.
- `POST /api/chat { roomId, userId, message }` → Persists a chat message and triggers SSE updates. Response `{ message: string }`.
- `GET /api/chat/{roomId}` → Historical chat transcript (`main.ChatMessage[]`).
- `POST /api/leave { userId: string }` → Removes the user from their room and may tear down the room. Response `{ message: string }`.
- `GET /api/operations/{roomId}?since=<opId>` → Returns git-style operations recorded after the provided operation ID so reconnecting clients can catch up before resuming SSE.

### Server-Sent Events

- `GET /api/sse?userId=<id>` → Opens an SSE stream for the user. The handler keeps the connection alive with 30s heartbeats and cleans up on disconnect.
- Event payloads are wrapped as `{ type, data, timestamp }`:
    - `connected` → `{ status: "connected" }`
    - `user_created` → `main.User`
    - `room_created` → `main.Room`
    - `room_deleted` → `{ roomId, roomName }`
    - `user_invited` → `{ roomId, roomName, inviter }`
    - `user_joined` → `{ roomId, roomName, userId, userName }`
    - `user_left` → `{ roomId, roomName, userId, userName }`
    - `chat_message` → `main.ChatMessage`
    - `clipboard_copied` → `{ type: 'text' | 'image', text?, image? }`
    - `heartbeat` → `{ timestamp }` (maintenance; emitted automatically)

## Core Data Structures

```typescript
export interface User {
    id: string;
    name: string;
    roomId?: string;
    isOnline: boolean;
}

export interface Room {
    id: string;
    name: string;
    userIds: string[];
}

export interface ChatMessage {
    id: string;
    roomId: string;
    userId: string;
    userName: string;
    message: string;
    timestamp: number; // Unix seconds
}
```

### REST DTOs

```jsonc
// POST /api/users
{ "name": string }

// POST /api/invite
{ "userId": string }

// POST /api/chat
{ "roomId": string, "userId": string, "message": string }

// POST /api/leave
{ "userId": string }

// POST /api/rooms
{ "name": string }

// Generic success envelope
{ "message": string }
```

## Quick Usage Example

```javascript
import { GetMode, ListAllUsers, Invite } from '../wailsjs/go/main/App';

const mode = await GetMode();

if (mode === 'host') {
    const users = await ListAllUsers();
    const firstUser = users.at(0);
    if (firstUser) {
        const result = await Invite(firstUser.id);
        console.log(result);
    }
}
```