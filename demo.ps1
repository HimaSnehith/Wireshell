# Wireshell One-Click Launch & Connect Test
Write-Host "⚡ Building & Launching Wireshell P2P Test..." -ForegroundColor Cyan

go build -ldflags="-s -w" -o wireshell.exe ./cmd/wireshell
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed." -ForegroundColor Red
    exit 1
}

.\wireshell.exe test-mesh
