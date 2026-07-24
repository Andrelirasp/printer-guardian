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

Write-Host "[1/3] Verificando ferramentas de recurso (rsrc)..." -ForegroundColor Yellow
$rsrcPath = "$env:USERPROFILE\go\bin\rsrc.exe"
if (-not (Get-Command rsrc -ErrorAction SilentlyContinue) -and -not (Test-Path $rsrcPath)) {
    Write-Host "Instalando rsrc automaticamente via 'go install'..." -ForegroundColor Yellow
    go install github.com/akavel/rsrc@latest
}

Write-Host "[2/3] Gerando arquivo de recursos com ícone e manifesto UAC (main.syso)..." -ForegroundColor Yellow
if (Test-Path "icon.ico") {
    if (Test-Path $rsrcPath) {
        if (Test-Path "manifest.xml") {
            & $rsrcPath -ico icon.ico -manifest manifest.xml -o main.syso
        } else {
            & $rsrcPath -ico icon.ico -o main.syso
        }
    } elseif (Get-Command rsrc -ErrorAction SilentlyContinue) {
        if (Test-Path "manifest.xml") {
            rsrc -ico icon.ico -manifest manifest.xml -o main.syso
        } else {
            rsrc -ico icon.ico -o main.syso
        }
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
