# Wireshell PowerShell Runner for Windows
param (
    [Parameter(Position=0)]
    [string]$Command = "help"
)

# Ensure binary is built
if (-not (Test-Path "wireshell.exe")) {
    Write-Host "Building wireshell.exe..." -ForegroundColor Cyan
    go build -ldflags="-s -w" -o wireshell.exe ./cmd/wireshell
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed." -ForegroundColor Red
        exit 1
    }
}

switch ($Command.ToLower()) {
    "connect" {
        Write-Host "Launching Wireshell VPN (Administrator privileges required to create network adapter)..." -ForegroundColor Cyan
        .\wireshell.exe connect
    }
    "ip" {
        .\wireshell.exe ip
    }
    "test" {
        .\wireshell.exe test-mesh
    }
    "build" {
        Write-Host "Rebuilding wireshell.exe..." -ForegroundColor Cyan
        go build -ldflags="-s -w" -o wireshell.exe ./cmd/wireshell
        Write-Host "Build finished successfully." -ForegroundColor Green
    }
    default {
        Write-Host "Wireshell Commands for Windows PowerShell:" -ForegroundColor Yellow
        Write-Host "  .\run.ps1 connect   -> Connect to encrypted WireGuard VPN (changes IP address)" -ForegroundColor Green
        Write-Host "  .\run.ps1 ip        -> Check your current public IP and location" -ForegroundColor Green
        Write-Host "  .\run.ps1 test      -> Run the P2P mesh & hole-punch test" -ForegroundColor Green
        Write-Host "  .\run.ps1 build     -> Rebuild the wireshell.exe binary" -ForegroundColor Green
    }
}

