# Test Clip macOS utility

This small utility watches the macOS clipboard and posts to the local HTTP endpoint in the app to trigger a visual overlay.

Quick start
1. Start the main Wails host application from the repo root (ensure it listens on 8080):
```
cd /Users/richard/Documents/programming/GO/GoTeamWork
go run main.go --mode host &
```

2. Run this clip watcher in the test_clip folder:
```
cd test_clip
go run .
```

3. Press `Cmd+C` anywhere (or use the app's copy actions). The watcher will detect the clipboard change and POST to /api/overlay which should trigger a small overlay icon in the app window for a short duration.

Note
- This project uses macOS pasteboard APIs, and you may see a deprecation warning for `NSFilenamesPboardType` in newer macOS SDKs; behaviour is still functional.
- The overlay is implemented in the Wails frontend and is shown by the host app via an `overlay:show` event.
