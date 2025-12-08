# Wails v2 → v3 Migration Progress (paused)

**Date:** 2025-12-08  
**Branch:** upgrade-wails3 (work paused here)

## What was done
- Updated `wails.json` to v3 schema with new frontend block.
- Swapped backend bootstrap to `application.New` and service pattern; added `ServiceStartup` hook in `app.go`.
- Regenerated bindings into `frontend/bindings` and pointed TS imports to them; installed `@wailsio/runtime` (3.0.0-alpha.74).
- Updated `tsconfig.json` include paths and refreshed package deps; ran `go mod tidy`.

## Known issues (build still failing)
- Window creation uses a non-existent v3 API (`NewWebviewWindowWithOptions`); needs the v3 `NewWindow` path.
- `clip.go` and other spots still reference the old `ctx` pattern; service holds `*application.Application` and emits events—needs refactor.
- Bindings generation logged missing go.sum entries (xdg, go-git, tint) and undefined `application.Application` errors; CGO warnings on darwin accessibility files remain.
- No successful `go build`/`wails3 build` yet after migration changes.

## Decision
Migration on `upgrade-wails3` is paused. Next steps will proceed on a new branch with a clean Wails 3 rebuild, starting by removing Wails-specific frontend and API layers.
