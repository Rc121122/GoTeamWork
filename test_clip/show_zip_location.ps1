# ZIP檔案位置演示
Write-Host "=== ZIP檔案位置演示 ===" -ForegroundColor Green
Write-Host ""

# 顯示當前目錄
Write-Host "當前程式目錄:" -ForegroundColor Yellow
Write-Host "  $(Get-Location)" -ForegroundColor Cyan
Write-Host ""

# 創建測試檔案
Write-Host "1. 創建測試檔案..." -ForegroundColor Yellow
"這是測試檔案內容" | Out-File -FilePath "demo_file.txt" -Encoding UTF8
Write-Host "   ✓ 創建 demo_file.txt" -ForegroundColor Green
Write-Host ""

# 模擬ZIP創建
Write-Host "2. 模擬ZIP檔案創建..." -ForegroundColor Yellow
$timestamp = [int](Get-Date -UFormat %s)
$zipName = "share_$timestamp.zip"
Write-Host "   單一檔案ZIP: $zipName" -ForegroundColor Cyan
Write-Host "   完整路徑: $(Get-Location)\$zipName" -ForegroundColor Cyan
Write-Host ""

$multiZipName = "shared_files_$timestamp.zip"
Write-Host "   多檔案ZIP: $multiZipName" -ForegroundColor Cyan
Write-Host "   完整路徑: $(Get-Location)\$multiZipName" -ForegroundColor Cyan
Write-Host ""

# 實際創建一個ZIP來演示
Write-Host "3. 實際創建ZIP檔案..." -ForegroundColor Yellow
Compress-Archive -Path "demo_file.txt" -DestinationPath $zipName -Force
if (Test-Path $zipName) {
    Write-Host "   ✓ 創建成功: $zipName" -ForegroundColor Green
    Write-Host "   📁 檔案位置: $(Get-Location)\$zipName" -ForegroundColor Green
    $size = (Get-Item $zipName).Length
    Write-Host "   📊 檔案大小: $([math]::Round($size/1KB, 2)) KB" -ForegroundColor Green
} else {
    Write-Host "   ✗ 創建失敗" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== 總結 ===" -ForegroundColor Green
Write-Host "ZIP檔案會在程式運行目錄中創建，方便你查看和管理！" -ForegroundColor Cyan
Write-Host "使用完畢後記得手動刪除ZIP檔案。" -ForegroundColor Yellow
Write-Host ""

# 清理演示檔案
Write-Host "清理演示檔案..." -ForegroundColor Gray
Remove-Item "demo_file.txt" -ErrorAction SilentlyContinue
Remove-Item $zipName -ErrorAction SilentlyContinue
Write-Host "   ✓ 清理完成" -ForegroundColor Green