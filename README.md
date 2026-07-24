# Printer Guardian v2.0

Agente Windows autônomo e silencioso que monitora, protege e corrige automaticamente problemas de impressoras USB, rede, Bluetooth e integrações com QZ Tray. Compilado em Go em um único executável `.exe` com ícone nativo, manifesto UAC e Dashboard Web de diagnóstico.

---

## 🚀 O que há de novo no Printer Guardian v2.0

| Funcionalidade | Descrição |
| --- | --- |
| 🎨 **Ícone Nativo e Profissional** | Executável compilado com ícone 3D embutido no arquivo de recursos (.syso) |
| 🔑 **Manifesto UAC (Administrador)** | Executa nativamente com permissões necessárias para gerenciar portas e serviços do Windows |
| 🪵 **Rotação Automática de Logs** | Limite máximo de **5 MB** por log. Rotaciona até 3 backups antigos automaticamente |
| 🩺 **Health Check Real do QZ Tray** | Teste HTTP ativo nas portas 8182/8181. Detecta e reinicia o QZ Tray mesmo se o Java estiver congelado (*deadlock*) |
| 🧹 **Autocura do Spooler do Windows** | Limpa spools corrompidos em `System32\spool\PRINTERS\*` e reinicia o serviço Spooler |
| 🌐 **Dashboard Web Local (`http://localhost:9123`)** | Mini servidor HTTP nativo com log ao vivo e botões de ação rápida (*"Carregar Log"*, *"Reiniciar QZ"*, *"Limpar Spooler"*) |
| 💻 **Compatibilidade Universal** | Suporte a Windows 7, 8, 8.1, 10 e 11 (com fallback WMI automático no Win 7) |

---

## 🛠️ Como Compilar em um Computador Novo (Zero Complicação)

Você **não precisa** de nenhuma IA ou comandos complexos para compilar o projeto em uma máquina nova.

### Pré-requisito Único
Instale o **Go** no computador:
👉 Baixar Go (Windows 64-bit): **[go.dev/dl/](https://go.dev/dl/)**

---

### Opção A: Pelo Windows Explorer (Um Clique)

1. Dê um **duplo clique no arquivo `build.bat`**.
2. O script fará tudo sozinho:
   - Baixará a ferramenta de ícone (`rsrc`) se necessário.
   - Gerará o ícone 3D e o manifesto UAC no executável.
   - Compilará o `PrinterGuardian.exe`.
3. Pronto! O executável estará pronto na mesma pasta.

---

### Opção B: Pelo PowerShell

No terminal da pasta do projeto, execute:

```powershell
.\build.ps1
```

---

### Opção C: Pelo Git Bash / Linux

```bash
chmod +x build.sh
./build.sh
```

---

## 🖥️ Dashboard Web Local de Diagnóstico

Quando o Printer Guardian estiver rodando, abra o navegador e acesse:

👉 **[http://localhost:9123](http://localhost:9123)**

![Dashboard Local](web/index.html)

No Dashboard você pode:
* 🟢 **Acompanhar o Log Ao Vivo** do cliente em tempo real.
* 🔄 **Reiniciar o QZ Tray** manualmente com 1 clique.
* 🧹 **Limpar o Spooler de Impressão** com 1 clique.
* 🔍 **Analisar falhas e portas USB** visualmente.

---

## ⚙️ Configuração via JSON (`config.json`)

O arquivo `config.json` fica na mesma pasta do executável. Se não existir, será gerado automaticamente.

```json
{
  "checkInterval": 15,
  "enableUSBFix": true,
  "enableSNMPFix": true,
  "enableBluetoothFix": true,
  "enableNewPrinterDetection": true,
  "enableSelfMonitoring": true,
  "enableQZTrayWatch": true,
  "autoMapPrinters": false,
  "whitelist": [],
  "blacklist": [],
  "printerMappings": [],
  "maintenanceMode": false
}
```

| Campo | Tipo | Descrição |
| --- | --- | --- |
| `checkInterval` | int | Intervalo em segundos entre verificações (padrão: 15) |
| `enableUSBFix` | bool | Habilita correção automática de portas USB |
| `enableSNMPFix` | bool | Habilita desativação automática de SNMP em portas de rede |
| `enableBluetoothFix` | bool | Habilita correção de impressoras Bluetooth |
| `enableQZTrayWatch` | bool | Habilita monitoramento e verificação de saúde HTTP do QZ Tray |
| `whitelist` | array | Lista de nomes de impressoras para processar (vazio = todas) |
| `blacklist` | array | Lista de nomes de impressoras para ignorar (ex: PDF Printers) |

---

## 📂 Arquivos Gerados em Tempo de Execução

| Arquivo | Descrição |
| --- | --- |
| `PrinterGuardian.exe` | Executável final com ícone e elevação de privilégios |
| `printer-guardian.log` | Log atual do agente (máximo 5 MB) |
| `printer-guardian.1.log` | Backup de log rotacionado |
| `backups/*.json` | Backup automático da configuração original das portas |

---

## 📦 Guia de Instalação no Cliente

1. Copie o `PrinterGuardian.exe` para a máquina do cliente (ex: `C:\PrinterGuardian\`).
2. Pressione `Win + R`, digite `shell:startup` e crie um atalho do `PrinterGuardian.exe` dentro desta pasta.
3. Dê duplo clique no `PrinterGuardian.exe`. Ele rodará em segundo plano protegendo todas as impressoras e a conexão com o QZ Tray.

---

## 📄 Licença

Projeto fornecido para produção e distribuição em ambientes comercial/SaaS.
