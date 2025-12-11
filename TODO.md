## ToDo:
1. Block same username creation, don't allow to create user if username exists.
2. HUD icon is not transparent in mac, search for online solution.
   suggest resource: https://github.com/wailsapp/wails/issues/3036

## Done:
1. Room Isolation: Implemented OwnerID and ApprovedUserIDs.
2. Join Permission: Added "Request to Join" flow and enforced permissions in JoinRoom.
3. Zip Download: Fixed regression where downloads failed due to room isolation (updated handleDownload to search all rooms).
4. Username Conflict: Added auto-renaming (e.g., "User (1)") to prevent duplicate names.