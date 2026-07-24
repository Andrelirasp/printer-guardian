# Script de Compilação Automática do Printer Guardian v2.0
Write-Host "====================================================" -ForegroundColor Cyan
Write-Host "  Printer Guardian v2.0 - Compilação Automática" -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERRO] O Go não está instalado neste computador!" -ForegroundColor Red
    Write-Host "Por favor, baixe e instale o Go em: https://go.dev/dl/" -ForegroundColor Yellow
    Exit 1
}

Write-Host "[1/3] Verificando ferramentas de recurso (go-winres)..." -ForegroundColor Yellow
$winresPath = "$env:USERPROFILE\go\bin\go-winres.exe"
if (-not (Get-Command go-winres -ErrorAction SilentlyContinue) -and -not (Test-Path $winresPath)) {
    Write-Host "Instalando go-winres automaticamente via 'go install'..." -ForegroundColor Yellow
    go install github.com/tc-hib/go-winres@latest
}

Write-Host "[2/3] Gerando arquivo de recursos com ícone 3D e UAC (rsrc_windows_amd64.syso)..." -ForegroundColor Yellow
if (Test-Path "winres\winres.json") {
    if (Test-Path $winresPath) {
        & $winresPath make
    } elseif (Get-Command go-winres -ErrorAction SilentlyContinue) {
        go-winres make
    }
}

Write-Host "[3/3] Compilando PrinterGuardian.exe..." -ForegroundColor Yellow
go build -ldflags="-H windowsgui" -o PrinterGuardian.exe main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "====================================================" -ForegroundColor Green
    Write-Host "  SUCESSO! Executável gerado: PrinterGuardian.exe" -ForegroundColor Green
    Write-Host "====================================================" -ForegroundColor Green
    Write-Host ""
} else {
    Write-Host "[ERRO] Falha na compilação." -ForegroundColor Red
}
