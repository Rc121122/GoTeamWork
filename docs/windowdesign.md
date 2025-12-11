### Wails2 only support a single window per application.
## solutions: using React VirutalDOM to change content dynamically.

---

## Design

### Main windows:
    Create a main window with a sidebar for navigation.
    Thinner sidebar with buttons to open pop up small windows. (DOM, not real new windows)
        - Reboot
            restart the app, go to landing page and select mode again.
        - Settings
            App settings
        - About
            Show app version, link to github repo
    Pages (client):
        - Landing Page (Home):
            support mode selection in future
        - New User Page:
            Enter username to create new user

        -> User after the above two steps complete initialization, can't go back except reboot.

        - Lobby:
            Show all other users and invite buttons
        - Room:
            chatroom at left column, shared clipboard at right column (use card layout)
        -> these two pages point to each other in FSM, if user leave room, go back to lobby.

    Pages (central server:host):
        Single page:
            Left column: list all users
            Right column: list all rooms, with user count.

### Overlay HUD:
    User click "ctrl+c/cmd+c" or "ctrl+v/cmd+v" system detection calls HUD on top of all windows.
    Only when the main window is non-active (User focus on other application)
    ctrl+c/cmd+c: display a cute golang gopher next to mouse cursor, click the gopher to share clipboard. (do not share without user permission)
    Select content from share clipboard, then press ctrl+v/cmd+v else where: display a gopher that carry a square (shared content) and throw to user cursor pasting position. 

---

### Questions (Added by Copilot)
1. **HUD Implementation**: Since Wails 2 (typically) supports a single window, how should the HUD be implemented? Should the main window resize and move to the cursor location, becoming transparent? Or is the intention to toggle the visibility/style of the single window to act as the HUD?
2. **Assets**: Do you have the specific "cute golang gopher" image assets (idle, carrying square, throwing) ready, or should placeholders be used?
3. **Host vs Client Mode**: The current application has distinct "Host" and "Client" modes. Should the new Sidebar/Navigation be available in both modes, or does the "Room Management" view handle the distinction (e.g., you choose to be Host or Client there)?
4. **System Global Hotkeys**: The HUD relies on detecting `ctrl+c` globally. Is the backend logic for global hotkey detection already implemented and exposing events to the frontend?

A1. Yes, the HUD is the same window of main window, change size, position and transparency when triggered ASAP to provide a smooth user experience.

A2. not ready, create a folder for me, "frontend/src/assets/gopher", use placeholder images for now.

A3. Keep it simple, use same navigation for both modes. Host and client are decided in start up. (Host is actually a central server, not involved in rooms, just viewing status, list all users/rooms) Later on we will build LAN mode, where there's no central server, it'll be discussed later.

A4. Yes, it's implemented in /test_clip folder, it is tested and working, just need to bridge to frontend.