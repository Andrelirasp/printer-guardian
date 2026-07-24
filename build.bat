@echo off
echo ====================================================
echo   Printer Guardian v2.0 - Compilacao Automatica
echo ====================================================
echo.

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERRO] O Go nao esta instalado neste computador!
    echo Por favor, baixe e instale o Go em: https://go.dev/dl/
    pause
    exit /b 1
)

echo [1/3] Verificando ferramentas de recurso (go-winres)...
where go-winres >nul 2>nul
if %errorlevel% neq 0 (
    if not exist "%USERPROFILE%\go\bin\go-winres.exe" (
        echo Instalando go-winres automaticamente...
        go install github.com/tc-hib/go-winres@latest
    )
)

echo [2/3] Gerando arquivo de recursos com icone 3D e UAC (rsrc_windows_amd64.syso)...
if exist winres\winres.json (
    if exist "%USERPROFILE%\go\bin\go-winres.exe" (
        "%USERPROFILE%\go\bin\go-winres.exe" make
    ) else (
        go-winres make
    )
)

echo [3/3] Compilando PrinterGuardian.exe...
go build -ldflags="-H windowsgui" -o PrinterGuardian.exe .

if %errorlevel% equ 0 (
    echo.
    echo ====================================================
    echo   SUCESSO! Executavel gerado: PrinterGuardian.exe
    echo ====================================================
    echo.
) else (
    echo.
    echo [ERRO] Falha ao compilar o executavel.
)
pause
