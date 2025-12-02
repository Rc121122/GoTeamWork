# 檔案分享功能測試腳本
# 使用方法: 在 PowerShell 中運行此腳本

Write-Host "=== 檔案分享功能測試 ===" -ForegroundColor Green
Write-Host ""

# 步驟 1: 編譯程式
Write-Host "1. 編譯程式..." -ForegroundColor Yellow
go build -o clipwindows.exe clipwindows.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✓ 編譯成功" -ForegroundColor Green
} else {
    Write-Host "   ✗ 編譯失敗" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 步驟 2: 運行測試模式
Write-Host "2. 運行功能測試..." -ForegroundColor Yellow
.\clipwindows.exe --test
Write-Host ""

# 步驟 3: 啟動監控模式
Write-Host "3. 啟動檔案監控模式..." -ForegroundColor Yellow
Write-Host "   程式正在背景運行，監控檔案複製..." -ForegroundColor Cyan
Write-Host "   現在你可以嘗試在檔案總管中複製檔案來測試功能" -ForegroundColor Cyan
Write-Host ""
Write-Host "按 Ctrl+C 停止程式" -ForegroundColor Gray

# 啟動程式（不會等待，因為它是監控模式）
Start-Process -NoNewWindow -FilePath ".\clipwindows.exe"