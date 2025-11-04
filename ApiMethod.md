# JavaScript API Methods

## Core Application Methods

### GetMode()
- **Description**: Returns the current application mode ('host' or 'client')
- **Returns**: `Promise<string>`
- **Usage**: `const mode = await GetMode()`

### Greet(name: string)
- **Description**: Returns a greeting message for the given name
- **Parameters**: `name` (string) - The name to greet
- **Returns**: `Promise<string>`
- **Usage**: `const greeting = await Greet("John")`

## User Management Methods

### ListAllUsers()
- **Description**: Returns a list of all users in the system
- **Returns**: `Promise<Array<main.User>>`
- **Usage**: `const users = await ListAllUsers()`

### CreateUser(name: string)
- **Description**: Creates a new user in the system
- **Parameters**: `name` (string) - The username for the new user
- **Returns**: `Promise<main.User>`
- **Usage**: `const user = await CreateUser("Alice")`

## Room Management Methods

### GetAllRooms()
- **Description**: Returns all rooms in the system
- **Returns**: `Promise<Array<main.Room>>`
- **Usage**: `const rooms = await GetAllRooms()`

### GetCurrentRoom()
- **Description**: Returns information about the current room
- **Returns**: `Promise<main.Room>`
- **Usage**: `const currentRoom = await GetCurrentRoom()`

### Invite(userID: string)
- **Description**: Invites a user to the current room. Creates a new room if none exists.
- **Parameters**: `userID` (string) - The ID of the user to invite
- **Returns**: `Promise<string>` - Success/error message
- **Usage**: `const result = await Invite("user_123")`

### LeaveRoom(userID: string)
- **Description**: Removes a user from their current room. Deletes room if < 2 users remain.
- **Parameters**: `userID` (string) - The ID of the user leaving
- **Returns**: `Promise<string>` - Success/error message
- **Usage**: `const result = await LeaveRoom("user_123")`

## Server Methods

### StartHTTPServer(port: string)
- **Description**: Starts the HTTP server for REST API (host mode only)
- **Parameters**: `port` (string) - The port number to listen on
- **Returns**: `Promise<void>`
- **Usage**: `await StartHTTPServer("8080")`

## Data Structures

### User
```typescript
interface User {
    id: string;        // Unique user identifier
    name: string;      // Display name
    roomId?: string;   // Current room ID (optional)
    isOnline: boolean; // Online status
}
```

### Room
```typescript
interface Room {
    id: string;      // Unique room identifier
    name: string;    // Display name
    userIds: string[]; // Array of user IDs in the room
}
```

## REST API Endpoints (Host Mode)

### GET /api/users
- **Description**: List all users
- **Response**: `Array<User>`

### POST /api/users
- **Description**: Create a new user
- **Body**: `{"name": "username"}`
- **Response**: `User`

### GET /api/users/{id}
- **Description**: Get specific user by ID
- **Response**: `User`

### GET /api/rooms
- **Description**: List all rooms
- **Response**: `Array<Room>`

### POST /api/invite
- **Description**: Invite a user to current room
- **Body**: `{"userId": "user_id"}`
- **Response**: `{"message": "result message"}`

## Usage Examples

```javascript
// Check application mode
const mode = await GetMode();
console.log('Running in', mode, 'mode');

// List all users
const users = await ListAllUsers();
users.forEach(user => {
    console.log(`${user.name} (${user.isOnline ? 'online' : 'offline'})`);
});

// Create a new user
const newUser = await CreateUser("Bob");
console.log('Created user:', newUser.name);

// Invite user to room
const inviteResult = await Invite(newUser.id);
console.log('Invite result:', inviteResult);

// Get current room info
const currentRoom = await GetCurrentRoom();
if (currentRoom) {
    console.log('Current room:', currentRoom.name);
    console.log('Users in room:', currentRoom.userIds.length);
}
```