#!/bin/bash

# Script de compilação para Windows com GUI invisível
# Uso: ./build.sh

set -e

echo "Compilando Printer Guardian para Windows..."

# Compilação cruzada para Windows com flag windowsgui
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o PrinterGuardian.exe main.go

if [ $? -eq 0 ]; then
    echo "✓ Compilação concluída com sucesso!"
    echo "  Executável gerado: PrinterGuardian.exe"
    echo ""
    echo "Para instalar no cliente:"
    echo "  1. Copie PrinterGuardian.exe para a pasta de inicialização do Windows"
    echo "  2. Pressione Win+R e digite: shell:startup"
    echo "  3. Cole o executável nesta pasta"
    echo "  4. O programa iniciará automaticamente com o Windows"
else
    echo "✗ Erro na compilação"
    exit 1
fi
