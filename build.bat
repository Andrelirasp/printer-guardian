@echo off
chcp 65001 > nul
echo ====================================================
echo   Printer Guardian v2.0 - Compilação Automática
echo ====================================================
echo.

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERRO] O Go não está instalado neste computador!
    echo Por favor, baixe e instale o Go em: https://go.dev/dl/
    pause
    exit /b 1
)

echo [1/3] Verificando ferramentas de recurso (rsrc)...
where rsrc >nul 2>nul
if %errorlevel% neq 0 (
    if not exist "%USERPROFILE%\go\bin\rsrc.exe" (
        echo Instalando rsrc automaticamente...
        go install github.com/akavel/rsrc@latest
    )
)

echo [2/3] Gerando arquivo de recursos com ícone e manifesto UAC (main.syso)...
if exist icon.ico (
    if exist "%USERPROFILE%\go\bin\rsrc.exe" (
        if exist manifest.xml (
            "%USERPROFILE%\go\bin\rsrc.exe" -ico icon.ico -manifest manifest.xml -o main.syso
        ) else (
            "%USERPROFILE%\go\bin\rsrc.exe" -ico icon.ico -o main.syso
        )
    ) else (
        if exist manifest.xml (
            rsrc -ico icon.ico -manifest manifest.xml -o main.syso
        ) else (
            rsrc -ico icon.ico -o main.syso
        )
    )
)

echo [3/3] Compilando PrinterGuardian.exe...
go build -ldflags="-H windowsgui" -o PrinterGuardian.exe main.go

if %errorlevel% equ 0 (
    echo.
    echo ====================================================
    echo   SUCESSO! Executável gerado: PrinterGuardian.exe
    echo ====================================================
    echo.
) else (
    echo.
    echo [ERRO] Falha ao compilar o executável.
)
pause
