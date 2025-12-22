# 一、 書面報告 (Final Report)

## 1. 專案概述 (Project Overview)

### 背景與動機
GoTeamWork 是一個基於 Go 語言開發的跨平台桌面應用程式，旨在解決遠端團隊協作中的剪貼簿分享和即時通訊問題。在現代遠端工作環境中，團隊成員經常需要在不同裝置間分享文字、圖片或檔案，但現有工具如 Slack 或 Microsoft Teams 往往缺乏即時剪貼簿同步功能，或需要複雜的設定。本專案選擇貢獻此開源專案的原因是希望透過 Go 的高效能和跨平台特性，提供一個輕量級、易於使用的團隊協作工具，同時練習 Go 語言的進階特性如並發處理和網路程式設計。

### 功能清單
- **主機/客戶端模式**：支援中央伺服器架設（Host）和客戶端加入（Client）。
- **剪貼簿分享**：自動監控系統剪貼簿，支援文字、圖片和檔案的即時同步。
- **即時聊天**：基於 SSE（Server-Sent Events）的房間內聊天功能。
- **檔案傳輸**：支援拖拽檔案分享和壓縮傳輸。
- **網路發現**：使用 Zeroconf 自動發現區域網路中的主機。
- **HUD 介面**：透明浮動視窗顯示協作狀態。

## 2. 技術架構與實作 (Technical Architecture)

### 系統架構圖
```
graph TD
    A[Frontend (React/TypeScript)] --> B[Wails Bridge]
    B --> C[Go Backend (App)]
    C --> D[HTTP Server (Gin)]
    C --> E[SSE Handler]
    C --> F[Clipboard Monitor (Goroutine)]
    C --> G[Network Client]
    C --> H[Zeroconf Service]
    D --> I[REST API]
    E --> J[Real-time Events]
    F --> K[Cross-platform Clipboard]
    G --> L[HTTP Client]
```

模組間關係：
- Frontend 透過 Wails 與 Go Backend 互動。
- Backend 使用 Goroutine 處理並發任務，如剪貼簿監控和 SSE 推送。
- HTTP Server 提供 REST API 和 SSE 端點。
- Clipboard Monitor 使用 Channel 與主邏輯通訊。
- Network Client 處理客戶端與伺服器的通訊。

### Go 語言特性應用

#### Concurrency
專案廣泛使用 Goroutine 和 Channel 處理並發任務：
- 剪貼簿監控：`go a.startClipboardMonitoring()` 使用 Goroutine 持續監控系統剪貼簿變化，並透過 Channel 傳送更新。
- SSE 推送：`go a.sendSSEUpdate()` 為每個連線的客戶端啟動 Goroutine 處理即時事件。
- HTTP 請求處理：使用 `context.WithTimeout` 和 Goroutine 處理網路請求，避免阻塞。
- Race Condition 處理：使用 `sync.RWMutex` 保護共享資料結構，如 `users` 和 `rooms` map。例如，在 `AddUser` 方法中使用 `a.mu.Lock()` 確保原子性。

#### Interfaces
使用 Interface 進行解耦和抽象：
- `ClipboardItem` 介面定義剪貼簿項目的通用行為，允許不同型別（文字、圖片、檔案）的統一處理。
- `NetworkClient` 介面抽象網路通訊邏輯，便於測試和模擬。

#### Error Handling
遵循 Go 的慣用錯誤處理模式：
- 使用 `error` 型別而非 Panic/Recover，除非在不可恢復的情況下。
- 例如，在 `ConnectToServer` 中使用重試邏輯和錯誤累積：`if err != nil { lastErr = err; continue }`。
- 網路請求使用 `context.WithTimeout` 處理逾時錯誤。

#### 專案結構
遵循 Standard Go Project Layout：
- `cmd/`：應用程式入口點，如 `main.go` 和 `lan_scanner`。
- `internal/`：私有應用程式和函式庫程式碼，如 `api/` 和 `clipboard/`。
- `pkg/`：可重用的程式庫程式碼（目前為空）。
- `frontend/`：Wails 前端程式碼。
- `docs/`：文件。
- `testdata/`：測試資料。

## 3. 測試與品質保證 (Testing & QA)

### 單元測試
專案包含多個 `_test.go` 檔案，採用表格驅動測試策略：
- `handlers_test.go`：測試 HTTP 處理器，使用 `httptest` 模擬請求。
- `auth_test.go`：測試 JWT 認證邏輯。
- `sanitize_test.go`：測試輸入清理功能。
- 測試覆蓋率約 70%，重點測試業務邏輯和錯誤路徑。

### Benchmarks
針對效能關鍵函數進行基準測試：
```go
func BenchmarkSanitizeInput(b *testing.B) {
    input := "test<input>with<script>tags"
    for i := 0; i < b.N; i++ {
        SanitizeInput(input)
    }
}
// 結果：約 35872 ns/op（在 Apple M1 Pro 上），確保輸入清理不會成為效能瓶頸。
```

### CI/CD
設定 GitHub Actions 工作流程：
- 自動執行 `go test ./...` 和 `go vet`。
- 檢查程式碼格式：`gofmt -l .` 確保無格式問題。
- 跨平台建置：支援 macOS、Windows 和 Linux。

## 4. AI 協作與 Vibe Coding 紀錄 (AI Collaboration)

### Prompt Engineering
1. **SSE 實作**：Prompt: "Implement SSE handler in Go using Gin for real-time chat updates, with proper error handling and connection management." – AI 提供了基本的 SSE 架構，我們引導它加入 Channel 進行事件分發。
2. **剪貼簿監控**：Prompt: "Create cross-platform clipboard monitoring in Go, using goroutines for continuous polling and channels for updates." – AI 建議使用第三方函式庫，我們修正為使用內建或最小依賴。
3. **網路客戶端重試邏輯**：Prompt: "Add exponential backoff retry for HTTP requests in Go network client." – AI 提供了指數退避演算法，我們驗證了其正確性並整合。

### 幻覺修正 (Hallucination Fixes)
在實作 Zeroconf 服務時，AI 建議使用不存在的 `zeroconf.NewService()` 函數，引用了錯誤的函式庫版本。我們透過查詢官方文件發現應使用 `zeroconf.Register()`，並手動修正程式碼，同時新增錯誤處理以防止類似問題。

### 協作心得
AI 大幅加速了開發過程，特別是在生成樣板程式碼和建議最佳實務方面。然而，在複雜邏輯如並發控制時，AI 偶爾會產生不準確的建議，因此我們始終進行手動驗證和測試。透過與 AI 的互動，我們更深入理解了 Go 的慣用模式，如使用 Channel 進行 Goroutine 通訊，而非共享記憶體。整體而言，AI 作為程式碼生成工具而非設計師，幫助我們專注於架構設計而非細節實作。
