# Wails v3 Setup & Workflow

## Prerequisites
- Go 1.24+
- Node 18+ (for Vite frontend)
- Wails v3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.41`)
- macOS users: ensure Xcode command line tools are installed. Linker may warn about SDK version mismatch; align Xcode/SDK if you want to suppress.

## Install deps
```sh
cd /Users/richard/Documents/programming/GO/GoTeamWork
# Go modules
go mod tidy
# Frontend deps
cd frontend && npm install
```

## Generate bindings (Go → TS)
Run whenever Go exports change:
```sh
cd /Users/richard/Documents/programming/GO/GoTeamWork
wails3 generate bindings
```
- Output: `frontend/bindings/GOproject/*.js`
- TS declarations: use `frontend/src/bindings.d.ts` (update if new methods appear).

## Frontend dev (Vite only)
```sh
cd frontend
npm run dev
```
- Serves at `http://localhost:5173`.

## Wails dev (backend + webview)
```sh
cd /Users/richard/Documents/programming/GO/GoTeamWork
wails3 dev
```
- Uses `frontend` build/dev commands from `wails.json`.
- Embedded assets come from `frontend/dist` in production; during dev, Wails proxies Vite.

## Production build
```sh
cd /Users/richard/Documents/programming/GO/GoTeamWork
wails3 build
```
- Builds Go backend and bundles `frontend/dist` from `npm run build`.

## Entry points
- Backend service: `app.go` (`ServiceStartup` hooks into Wails).
- Window bootstrap: `main.go` creates window via `app.Window.NewWithOptions` and embeds assets from `frontend/dist`.
- Frontend: `frontend/src/main.ts` imports bindings from `frontend/bindings/GOproject/app.js`.

## Troubleshooting
- If TS complains about missing declarations, re-run `wails3 generate bindings` and refresh `frontend/src/bindings.d.ts`.
- Linker warnings about macOS SDK version are informational; update SDK/Xcode to silence.
- If Vite cannot find bindings, ensure `frontend/bindings` exists (regenerate) and `tsconfig.json` includes `bindings/**/*`.
