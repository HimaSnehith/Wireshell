# Wireshell Windows Build Script
Write-Host "🔨 Building Wireshell for Windows (amd64)..." -ForegroundColor Cyan

$env:GOOS = "windows"
$env:GOARCH = "amd64"

go build -ldflags="-s -w" -o wireshell.exe ./cmd/wireshell

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ wireshell.exe built successfully!" -ForegroundColor Green
    Write-Host "Run .\wireshell.exe --help to get started." -ForegroundColor Yellow
} else {
    Write-Host "❌ Build failed." -ForegroundColor Red
}

