#!/bin/bash

# Script de compilação para Windows com GUI invisível
# Uso: ./build.sh

set -e

echo "Compilando Printer Guardian para Windows..."

# Gera o arquivo de recursos Windows (.syso) contendo o ícone caso o rsrc esteja disponível
if command -v rsrc &> /dev/null && [ -f "icon.ico" ]; then
    echo "Gerando recursos do executável com ícone (main.syso)..."
    rsrc -ico icon.ico -o main.syso
fi

# Compilação para Windows com flag windowsgui (sem janela de terminal)
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
