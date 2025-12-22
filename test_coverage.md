# Test Coverage Snapshot (2025-12-23)

Command used:

```
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Summary
- Overall: ~50.2%
- Package `GOproject`: 49.4%
- Package `clip_helper`: 62.2%
- Package `cmd/lan_scanner`: 37.2%

## Package Details
### GOproject
- Well covered: core operations (`AddOperation`, `CreateRoom`, `GetCurrentClipboardItems`), network client helpers (`UploadZipData`, `UploadSingleFileData`), SSE manager, sanitizers, JWT helpers.
- Gaps: app lifecycle ([startup](app.go#L423), [shutdown](app.go#L448)), background cleanup ([startCleanupTasks](app.go#L905)), auth token validation ([authenticateToken](app.go#L1257)), HTTP server bootstrap ([StartHTTPServer](app.go#L1329)), join approval flows ([ApproveJoinRequest](app.go#L1392)), and many clipboard runtime paths ([StartClipboardMonitor](clip.go#L43), [broadcastClipboardUpdate](clip.go#L430), [StartClipboardWatcher](clip.go#L487), [ShareSystemClipboard](clip.go#L652), [GetClipboardItem](clip.go#L684)). Handlers with 0%: [handleUserByID](handlers.go#L79), [handleJoinRoom](handlers.go#L332), [handleJoinRequest](handlers.go#L378), [handleApproveJoin](handlers.go#L423), [handleClipboardUpload](handlers.go#L678). `main` remains unhit.

### clip_helper
- Covered: archive helpers (`TarPathsWithProgress`, `TarPathsFast`, `ZipFiles`, `UnzipData`), size formatting.
- Gaps: non-progress tar path (`TarPaths`), lower-level tar adders (`addPathToTar`), macOS-specific accessibility and clipboard paths ([HasAccessibilityPermission](clip_helper/accessibility_darwin.go#L30), [RequestAccessibilityPermission](clip_helper/accessibility_darwin.go#L34), [ReadClipboard](clip_helper/clipboard_darwin.go#L13), [getFilePathsFromPasteboard](clip_helper/pasteboard_darwin.go#L71)).

### cmd/lan_scanner
- Covered: subnet parsing and scanner helpers.
- Gaps: CLI `main` path untested.

## Next Steps
- Add integration-style tests for app startup/shutdown and HTTP bootstrap to lift zeroed paths.
- Exercise clipboard workflows end-to-end, including SSE `handleSSE` and clipboard upload handlers.
- Add platform-conditioned tests (macOS stubs) or mark as `//go:build`-specific to avoid 0% on unsupported platforms.
- Cover LAN scanner `main` with an argument-driven test harness.

Re-run `go test -coverprofile=coverage.out ./...` and `go tool cover -func=coverage.out` after changes to refresh this snapshot.
