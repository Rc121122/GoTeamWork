# GoTeamWork 期末專題報告

## 摘要

本專題報告介紹了 GoTeamWork 專案的開發過程，這是一個基於 Go 語言的團隊協作應用程式，支援跨裝置的剪貼簿同步與即時聊天功能。專案採用 Wails 框架整合 Go 後端與 TypeScript/React 前端，實現了主機模式（中央伺服器）與客戶端模式（使用者介面）的雙重架構。報告將詳細描述學習 Go 語言的歷程，以及專案中運用到的 Go 核心技巧，包括並發程式設計、網路處理、資料結構管理等關鍵技術。

## 學習 Go 的歷程

### 學習動機與初期準備

在開始此專案前，我對 Go 語言並無深入了解，主要程式設計經驗集中在 Python 與 JavaScript 上。選擇 Go 作為專案語言的主要原因是其在系統程式設計與網路應用上的優異表現，特別是其內建的並發支援與高效能特性。學習過程從官方文檔《The Go Programming Language》開始，逐步掌握基本語法、變數宣告、函數定義等基礎概念。

### 核心概念學習階段

學習 Go 的第一階段聚焦於語言的核心特性：

1. **變數與型別系統**：理解 Go 的靜態型別系統、零值概念，以及簡潔的變數宣告語法（`:=`）。特別是學習了 Go 的基本型別（int、string、bool）與複合型別（slice、map、struct）。

2. **函數與方法**：掌握 Go 的函數定義語法、多返回值特性，以及方法接收器（value receiver vs pointer receiver）的區別。

3. **錯誤處理**：學習 Go 獨特的錯誤處理模式，使用 `error` 介面與 `defer`、`panic`、`recover` 機制。

### 進階技巧掌握

隨著專案開發深入，學習重點轉向 Go 的進階特性：

1. **並發程式設計**：Goroutines 與 channels 是學習的重點。通過實作 SSE（Server-Sent Events）與網路同步功能，掌握了 goroutines 的建立、channels 的使用模式（unbuffered/buffered），以及 select 語句的應用。

2. **介面與多型**：學習 Go 的隱式介面實現，通過 `io.Reader`、`http.Handler` 等標準介面的使用，理解了組合優於繼承的設計哲學。

3. **記憶體管理**：理解 Go 的垃圾回收機制，學習使用指標（pointers）進行高效能操作，特別是在處理大型資料結構時。

### 實戰應用與問題解決

在專案開發過程中，通過實際編碼解決了多個技術挑戰：

- **HTTP 伺服器實作**：學習使用 `net/http` 套件建立 REST API，掌握路由處理、中介軟體（middleware）的設計模式。

- **JSON 處理**：熟練使用 `encoding/json` 套件進行資料序列化與反序列化。

- **同步機制**：通過 `sync.Mutex`、`sync.RWMutex` 等同步原語解決並發存取問題。

- **嵌入資源**：學習使用 `//go:embed` 指令嵌入前端資源。

學習過程中最大的挑戰是從指令式程式設計轉向 Go 的慣用模式，特別是錯誤處理與介面導向設計。通過不斷重構程式碼，逐步掌握了 Go 的「idiomatic」寫法。

## 專案介紹

### 專案背景與目標

GoTeamWork 是一個跨平台團隊協作工具，旨在提供即時剪貼簿同步與群組聊天功能。專案支援兩種運作模式：

- **主機模式（Host Mode）**：作為中央伺服器，提供 REST API 進行使用者與房間管理，同時支援剪貼簿同步與聊天介面。
- **客戶端模式（Client Mode）**：連接到主機伺服器，提供使用者驗證、大廳等待、即時更新等功能。

### 系統架構

專案採用模組化設計，主要組件包括：

- `main.go`：應用程式入口點，負責模式選擇、權限檢查、熱鍵設定。
- `app.go`：核心應用邏輯，使用者/房間管理、HTTP API 實作。
- `handlers.go`：HTTP 請求處理器，包含所有 REST API 端點。
- `sse.go`：Server-Sent Events 實作，處理即時通訊。
- `network.go`：網路操作，包含中央伺服器通訊與 LAN 發現。
- `clip_helper/`：跨平台剪貼簿操作模組。
- `frontend/`：TypeScript/React 前端介面。

### 主要功能

1. **即時剪貼簿同步**：支援文字、圖片等多媒體內容的跨裝置同步。
2. **群組聊天功能**：房間內即時訊息交換。
3. **使用者管理**：唯一使用者名稱驗證、線上狀態追蹤。
4. **房間系統**：動態房間建立、邀請機制、自動生命週期管理。
5. **跨平台支援**：使用 Wails 框架支援 macOS、Windows、Linux。

## 技術實現與 Go 核心技巧

### 並發程式設計（Goroutines 與 Channels）

GoTeamWork 大量運用 goroutines 實現並發處理：

```go
// SSE 事件廣播使用 goroutines
go func() {
    for event := range a.sseManager.eventChan {
        a.sseManager.broadcastEvent(event)
    }
}()

// sse.go lines 46-50, 31-35
type SSEManager struct {
    clients map[string]*SSEClient
    mu      sync.RWMutex
}

type SSEEvent struct {
    Type      SSEEventType `json:"type"`
    Data      interface{}  `json:"data"`
    Timestamp int64        `json:"timestamp"`
}
```

**技巧應用**：
- 使用 channels 進行 goroutines 間通訊
- select 語句處理多個 channel 操作
- context.Context 實現操作取消

### 網路程式設計（HTTP Server 與 REST API）

專案實作了完整的 HTTP 伺服器：

```go
func (a *App) startup(ctx context.Context) {
    // 啟動 HTTP 伺服器
    go func() {
        mux := http.NewServeMux()
        a.setupRoutes(mux)
        server := &http.Server{Addr: ":8080", Handler: mux}
        server.ListenAndServe()
    }()
}

// handlers.go lines 16-35
func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    
    if r.Method == "POST" {
        var req CreateUserRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
        // ... 處理請求
    }
}
```

**技巧應用**：
- `net/http` 套件建立 RESTful API
- JSON 編碼/解碼處理請求響應
- CORS 設定支援跨域請求
- 中介軟體模式處理共用邏輯

### 網路客戶端與重試機制

實現與中央伺服器的通訊，包含重試邏輯：

```go
// network.go lines 20-25, 39-70
type NetworkClient struct {
    serverURL  string
    httpClient *http.Client
    connected  bool
    mu         sync.RWMutex
}

func (n *NetworkClient) ConnectToServer() error {
    const maxRetries = 3
    const retryDelay = 2 * time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        req, err := http.NewRequest("GET", n.serverURL+"/api/users", nil)
        ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
        req = req.WithContext(ctx)
        resp, err := n.httpClient.Do(req)
        cancel()
        
        if err == nil && resp.StatusCode == http.StatusOK {
            n.connected = true
            return nil
        }
        
        if attempt < maxRetries {
            time.Sleep(retryDelay)
        }
    }
    return fmt.Errorf("failed to connect after retries")
}
```

**技巧應用**：
- context.Context 控制請求超時
- 指數退避重試策略
- 同步保護連線狀態

### 資料結構與同步機制

使用 map 與 slice 管理動態資料，並配合同步原語：

```go
// app.go lines 301-320
type App struct {
    users    map[string]*User
    rooms    map[string]*Room
    mu       sync.RWMutex
    // ...
}

// types.go lines 11-16, 19-25
type User struct {
    ID       string  `json:"id"`
    Name     string  `json:"name"`
    RoomID   *string `json:"roomId,omitempty"` // nil if not in any room
    IsOnline bool    `json:"isOnline"`
}

type Room struct {
    ID              string   `json:"id"`
    Name            string   `json:"name"`
    OwnerID         string   `json:"ownerId"`
    UserIDs         []string `json:"userIds"`
    ApprovedUserIDs []string `json:"approvedUserIds"`
}
```

**技巧應用**：
- `sync.RWMutex` 區分讀寫鎖提高並發效能
- 記憶體池模式管理歷史記錄
- 垃圾回收優化記憶體使用

### 操作歷史與雜湊鏈

實現類似 Git 的操作歷史追蹤：

```go
// types.go lines 69-79
type Operation struct {
    ID         string        `json:"id"`
    ParentID   string        `json:"parentId"`
    ParentHash string        `json:"parentHash,omitempty"`
    Hash       string        `json:"hash"`
    OpType     OperationType `json:"opType"`
    ItemID     string        `json:"itemId"`
    Item       *Item         `json:"item,omitempty"`
    Timestamp  int64         `json:"timestamp"`
    UserID     string        `json:"userId,omitempty"`
    UserName   string        `json:"userName,omitempty"`
}

// app.go lines 50-80
func (hp *HistoryPool) AddOperation(roomID string, opType OperationType, itemID string, item *Item, userID, userName string) *Operation {
    hp.mu.Lock()
    defer hp.mu.Unlock()
    
    // 計算操作雜湊
    hash := computeOperationHash(parentHash, opType, itemID, item, userID, userName, timestamp)
    
    op := &Operation{
        ID:         fmt.Sprintf("op_%d", hp.counter),
        ParentID:   parentID,
        ParentHash: parentHash,
        Hash:       hash,
        // ... 其他欄位
    }
    
    hp.operations[roomID] = append(hp.operations[roomID], op)
    hp.enforceLimits(roomID)
    return op
}
```

**技巧應用**：
- SHA256 雜湊確保操作完整性
- 父子鏈結構追蹤操作歷史
- 記憶體限制防止無限增長

### 跨平台剪貼簿操作

專案實現了跨平台的剪貼簿讀寫功能：

```go
// clip_helper/clipboard.go lines 17-28
type ClipboardItem struct {
    Type    ClipboardItemType `json:"type"`
    Text    string            `json:"text,omitempty"`
    Image   []byte            `json:"image,omitempty"`
    Files   []string          `json:"files,omitempty"`
    // ... 其他欄位
}

// clip_helper/clipboard_darwin.go lines 13-50
func ReadClipboard() (*ClipboardItem, error) {
    // 初始化剪貼簿
    err := clipboard.Init()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
    }

    // 優先檢查檔案
    if filePaths := getFilePathsFromPasteboard(); len(filePaths) > 0 {
        return &ClipboardItem{
            Type:  ClipboardFile,
            Files: filePaths,
            Text:  fmt.Sprintf("%d files selected", len(filePaths)),
        }, nil
    }

    // 檢查圖片
    if imgData := clipboard.Read(clipboard.FmtImage); len(imgData) > 0 {
        return &ClipboardItem{
            Type:  ClipboardImage,
            Image: imgData,
        }, nil
    }

    // 檢查文字
    if textData := clipboard.Read(clipboard.FmtText); len(textData) > 0 {
        return &ClipboardItem{
            Type: ClipboardText,
            Text: string(textData),
        }, nil
    }
    // ...
}
```

**技巧應用**：
- 平台特定的實現檔案（darwin, windows, other）
- 統一的資料結構抽象不同剪貼簿內容
- 檔案、圖片、文字等多媒體支援

### 錯誤處理與資源管理

Go 的慣用錯誤處理模式：

```go
// app.go lines 747-767
func (a *App) CreateUser(name string) *User {
    a.mu.Lock()
    defer a.mu.Unlock()

    cleanName := sanitizeUserName(name)
    if cleanName == "" {
        cleanName = fmt.Sprintf("User %d", a.userCounter+1)
    }

    // 產生使用者 ID
    a.userCounter++
    userID := fmt.Sprintf("user_%d", a.userCounter)
    user := &User{
        ID:   userID,
        Name: cleanName,
    }
    a.users[user.ID] = user
    return user
}
```

**技巧應用**：
- defer 語句確保資源清理
- 互斥鎖保護共享狀態
- 輸入清理與預設值處理

### 應用程式入口與框架整合

專案使用 Wails 框架整合 Go 後端與前端，實現跨平台桌面應用：

```go
// main.go lines 14-55
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	mode := flag.String("mode", "client", "Mode: 'host' for central-server host or 'client' for central-server client")
	flag.Parse()

	app := NewApp(*mode)

	err := wails.Run(&options.App{
		Title:  "GOproject",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Frameless:        true,
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})
}
```

**技巧應用**：
- `//go:embed` 指令編譯時嵌入前端資源
- Wails 框架提供 Go 與 JavaScript 互操作
- 命令列參數解析選擇運作模式

## 結論

通過 GoTeamWork 專案的開發，我不僅掌握了 Go 語言的核心特性，還深入理解了現代系統程式設計的實務技巧。專案成功整合了並發處理、網路通訊、跨平台支援等多項關鍵技術，驗證了 Go 在團隊協作工具開發上的適用性。

學習 Go 的過程讓我認識到，語言的簡潔設計背後蘊含著深厚的工程智慧。從 goroutines 的輕量並發到介面的隱式實現，每個特性都體現了實用主義的設計哲學。未來，我將繼續深化對 Go 的理解，探索更多在分散式系統、雲端服務等領域的應用。

專案開發過程中遇到的挑戰，如跨平台相容性、即時通訊效能等，都通過 Go 的強大標準庫與生態系統得到有效解決。這不僅提升了我的程式設計能力，也培養了系統性思考與問題解決的技能。

## 參考資料

1. Go 官方文檔：https://go.dev/doc/
2. Wails 框架文檔：https://wails.io/docs/introduction
3. 《The Go Programming Language》書籍
4. GoTeamWork 專案原始碼與文檔

---

報告完成日期：2025年12月12日  
專案版本：v1.0.0  
開發環境：Go 1.25.2, Wails v2, macOS 15.1.1