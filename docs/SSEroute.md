1. New user
Event: A new user join server
Route: User A join server -> Server handler -> add in user list -> SSE broadcast to all users

2. Invite
Event: A user invite another user to join room
Route: User A invite User B -> Server handler(Sanitizer for invite message) -> send invite event to User B via SSE -> User B accept/decline
User B accept -> Server handler -> add User B to room of User A/ if no room, create new room for User A & B -> SSE broadcast to all room members

3. Message/clipboard
Event: A user send message/clipboard to room
Route: User A send message/clipboard -> Server handler(Sanitizer for message/clipboard) -> save to DB(git style) -> SSE broadcast update (no full history) to all room members

4. Leave room
Event: A user leave room
Route: User A leave room -> Server handler -> remove User A from room -> if room empty, delete room -> SSE broadcast update to all room members

5. Reconnect
Event: A user reconnect after some time
Route: User A request for update (with git hash) -> Server handler -> check DB for updates since last hash -> send updates to User A -> SSE resume updates

