# Clarification Questions

1. **Room Creation Flow**: You mentioned "only user can invite user to create room". Does this mean a room is *automatically* created when I invite someone? Or do I explicitly click "Create Room" first, and then invite people into it?
2. **Invite Mechanism**: How does the invitation reach the other user?
    - In **LAN mode**: Is it a UDP broadcast that pops up on their screen?
    - In **Central mode**: Is it a notification via the server?
    - Does the invited user have to "Accept" the invite?
3. **Host Username**: Does the user starting the "Central/Host" mode also need to enter a username? (I will assume **Yes** for consistency unless specified otherwise).
4. **Clipboard Sharing**: "Share clipboard on right". Does this mean a list of clipboard history items that I can click to copy? Or a real-time view of the current clipboard content?
5. **"Pear" Mode**: You mentioned "LAN/host, pear". I assume you meant "Peer". In LAN mode, is the UI identical for the initial host and the subsequent peers?

# Answers to Clarification Questions
A1. A room can be created only and only if the user invites another user. There is no separate "Create Room" button.

A2. In central mode, its simple and already implemented by SSE(sever side events). 
In LAN mode, I'm not decided yet, but I had built lan_scanner.go. Plan to scan other users with that, then hit a specific port to invite them. (maybe :8080) After inside a room, other logic are the same: 1 host manage and send SSE to other users when update happens.
Yes, invited user needs to accept the invite, its done in central mode, but LAN mode is little tricky, help me with it.

A3. No, central/host acts like a manager and monitor, doesn't attend the whole server activity.

A4. Yes, its a list of clipboard history items. (Follow with git style which is partially implemented in go)

A5. typo: pear->peer. Yes, UI is identical for host and peer in LAN mode. But in golang logic, host act like a server and sent other user through SSE like in central/host mode.