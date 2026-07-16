# Printer Guardian

Agente Windows invisível que monitora e corrige automaticamente problemas de impressoras USB e de rede. Compilado em Go como um único `.exe` sem dependências, ideal para distribuição em massa em ambientes SaaS.

---

## Guia do Projeto

### O Problema

Em estabelecimentos comerciais (restaurantes, lojas, bares), impressoras térmicas conectadas por USB ou rede sofrem dois problemas recorrentes:

1. **Porta USB morta** - O cliente troca a impressora de porta USB no computador. O Windows cria uma nova porta (ex: `USB002` vira `USB003`) mas a fila de impressão continua apontando para a porta antiga. A impressora fica "Offline".
2. **Falso Offline por SNMP** - Em impressoras de rede, o protocolo SNMP reporta falsos status de "Offline" ao Windows, travando a fila de impressão.

### A Solução

O **Printer Guardian** roda invisível em background e a cada 15 segundos:

1. Varre todas as impressoras USB do sistema
2. Se alguma estiver Offline, Error, Unknown, Paused ou qualquer estado problemático, testa todas as portas USB disponíveis até encontrar a correta
3. Varre impressoras Bluetooth (portas `COM*` / `BTHPRINT:*`) e tenta reativá-las via PnP ou reinício do Spooler
4. Varre todas as portas de rede TCP/IP e desativa o SNMP se estiver causando problemas
5. Reinicia o serviço Spooler quando necessário
6. Notifica o usuário sobre correções aplicadas e também quando não consegue corrigir automaticamente
7. Monitora o QZ Tray e o reabre automaticamente se fechar

Tudo isso sem abrir nenhuma janela, sem precisar configurar nomes de impressoras, e funcionando com 1 ou mais impressoras simultaneamente.

### Funcionalidades

| Funcionalidade | Descrição |
| --- | --- |
| **Correção USB** | Detecta impressoras USB em qualquer estado problemático (Offline, Unknown, Paused, Error) e reatribui portas automaticamente |
| **Correção Bluetooth** | Detecta impressoras Bluetooth desconectadas (`COM*`/`BTHPRINT:*`) e tenta reativá-las via PnP ou reinício do Spooler |
| **Correção SNMP** | Desativa SNMP em portas de rede que causam falsos "offline" |
| **Notificações Completas** | Alertas de correção bem-sucedida, falha de correção, impressora fora de alcance e QZ Tray reiniciado |
| **Monitoramento QZ Tray** | Detecta se o QZ Tray fechou e o reabre automaticamente |
| **Logging Detalhado** | Registro com timestamp e status exato de cada impressora a cada ciclo |
| **Configuração Externa** | Arquivo `config.json` para ajustar comportamento sem recompilar |
| **Backup/Rollback** | Salva configuração de porta antes de modificar cada impressora |
| **Detecção de Novas Impressoras** | Identifica impressoras recém-conectadas automaticamente |
| **Self-Monitoring** | Verifica saúde do próprio serviço e recria recursos se necessário |
| **Execução Invisível** | Zero janelas ou telas pretas, roda 100% em background |

### Como Funciona

```text
                         Loop a cada 15s
                               |
          +----------+---------+---------+----------+
          |          |                   |          |
   USB Printers   BT Printers       Rede TCP/IP   QZ Tray
          |          |                   |          |
  Status OK?   Status OK?         SNMP ativo?  Rodando?
   |    |        |    |             |      |     |    |
  Sim  Não      Sim  Não           Sim   Não   Sim  Não
   |    |        |    |             |      |     |    |
  Skip Listar  Skip Tentar PnP  Desativ. Skip  Skip Abrir
       Portas       + Spooler    SNMP
          |          |
      Corrigiu?  Corrigiu?
       |    |     |    |
      Sim  Não  Sim  Não
       |    |    |    |
     Log  Alerta Log Alerta
     Notif     Notif  "Fora de
                      Alcance"
```

### Estrutura do Projeto

```text
auto-printerport/
  main.go          # Código fonte principal (Go)
  snmp.go          # Probe SNMP minimalista em Go puro
  go.mod           # Módulo Go (sem dependências externas)
  config.json      # Configuração de exemplo
  build.sh         # Script de compilação para Windows
  README.md        # Este arquivo
```

### Compilação (Para Desenvolvedores)

**Requisitos:** Go 1.21+ instalado.

```bash
# Linux/macOS
sudo apt install golang   # Ubuntu/Debian
brew install go           # macOS

# Compilar para Windows (executável invisível)
chmod +x build.sh
./build.sh

# Ou manualmente
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o PrinterGuardian.exe main.go
```

A flag `-H windowsgui` remove a janela de console. O executável roda 100% invisível no Gerenciador de Tarefas.

### Configuração via JSON

O arquivo `config.json` deve ficar na mesma pasta do executável. Se não existir, será criado automaticamente com valores padrão na primeira execução.

```json
{
  "checkInterval": 15,
  "enableUSBFix": true,
  "enableSNMPFix": true,
  "enableBluetoothFix": true,
  "enableNewPrinterDetection": true,
  "enableSelfMonitoring": true,
  "enableQZTrayWatch": true,
  "whitelist": [],
  "blacklist": [],
  "maintenanceMode": false
}
```

| Campo | Tipo | Descrição |
| --- | --- | --- |
| `checkInterval` | int | Intervalo em segundos entre verificações (padrão: 15) |
| `enableUSBFix` | bool | Habilita correção automática de portas USB |
| `enableSNMPFix` | bool | Habilita desativação automática de SNMP |
| `enableBluetoothFix` | bool | Habilita correção e monitoramento de impressoras Bluetooth |
| `enableNewPrinterDetection` | bool | Habilita detecção de impressoras recém-conectadas |
| `enableSelfMonitoring` | bool | Habilita verificação de saúde do serviço |
| `enableQZTrayWatch` | bool | Habilita monitoramento e reinício automático do QZ Tray |
| `whitelist` | array | Lista de nomes de impressoras para processar (vazio = todas) |
| `blacklist` | array | Lista de nomes de impressoras para ignorar |
| `maintenanceMode` | bool | Pausa todas as correções quando `true` |

#### Exemplos

**Processar apenas impressoras específicas:**

```json
{
  "whitelist": ["Impressora Caixa", "Impressora Cozinha"],
  "blacklist": []
}
```

**Ignorar impressoras virtuais:**

```json
{
  "whitelist": [],
  "blacklist": ["PDF Creator", "Microsoft Print to PDF"]
}
```

### Arquivos Gerados em Tempo de Execução

| Arquivo | Descrição |
| --- | --- |
| `printer-guardian.log` | Log de todas as ações com timestamp |
| `backups/*.json` | Backup da porta original de cada impressora antes de modificar |
| `config.json` | Criado automaticamente se não existir |

---

## Guia de Instalação no Cliente

Este guia é destinado a quem vai instalar o Printer Guardian no computador do cliente (o PC que tem as impressoras conectadas).

### Pré-requisitos

- Windows 7 ou superior (funciona em 8, 10 e 11)
- PowerShell (já incluso no Windows)
- Não precisa instalar nada mais (sem Java, sem Node, sem Python)

### Passo a Passo

#### 1. Copiar os Arquivos

Você precisa de apenas **2 arquivos**:

- `PrinterGuardian.exe` (o executável compilado)
- `config.json` (opcional, será criado automaticamente)

Copie ambos para uma pasta fixa no computador do cliente. Sugestão:

```text
C:\PrinterGuardian\
  PrinterGuardian.exe
  config.json
```

#### 2. Configurar Inicialização Automática

Para que o programa inicie automaticamente quando o Windows ligar:

1. Pressione `Win + R` no teclado
2. Digite `shell:startup` e pressione Enter
3. A pasta de inicialização do Windows vai abrir
4. Crie um **atalho** do `PrinterGuardian.exe` nesta pasta
   - Clique direito > Novo > Atalho
   - Aponte para `C:\PrinterGuardian\PrinterGuardian.exe`
   - Nomeie como "Printer Guardian"

A partir de agora, toda vez que o PC ligar, o Printer Guardian inicia automaticamente.

#### 3. Primeira Execução

Dê duplo clique no `PrinterGuardian.exe`. Nada vai aparecer na tela (isso é normal, ele roda invisível). Para confirmar que está rodando:

1. Abra o **Gerenciador de Tarefas** (`Ctrl + Shift + Esc`)
2. Na aba **Processos**, procure por `PrinterGuardian.exe`
3. Se estiver na lista, está funcionando

O arquivo `printer-guardian.log` será criado na mesma pasta do executável. Você pode abrir este arquivo com o Bloco de Notas para ver o histórico de ações.

#### 4. Testar o Funcionamento

Para testar se o Printer Guardian está realmente corrigindo portas:

1. Desconecte o cabo USB da impressora
2. Conecte em uma porta USB **diferente** no computador
3. Aguarde até 15 segundos
4. A impressora deve voltar ao status "Online" automaticamente
5. Uma notificação do Windows aparecerá confirmando a correção

#### 5. Parar o Programa (Se Necessário)

1. Abra o **Gerenciador de Tarefas**
2. Encontre `PrinterGuardian.exe` na lista de processos
3. Clique em **Finalizar Tarefa**

Para impedir que inicie com o Windows, remova o atalho da pasta `shell:startup`.

### Cenários Comuns

| Cenário | O que o Printer Guardian faz |
| --- | --- |
| Cliente troca impressora USB de porta | Detecta offline/unknown, testa portas, reatribui automaticamente |
| Impressora USB fica com status "Unknown" | Detecta e tenta corrigir (não só Offline/Error) |
| Impressora não corrige após tentar todas as portas | Notifica o usuário para verificar manualmente |
| Impressora Bluetooth desconecta | Tenta reativar via PnP e Spooler; notifica se fora de alcance |
| Cliente adiciona nova impressora | Detecta e registra no log |
| Impressora de rede fica "Offline" sem motivo | Desativa SNMP e reinicia o Spooler |
| QZ Tray fecha inesperadamente | Detecta e reabre automaticamente |
| Múltiplas impressoras (caixa + cozinha + bar) | Monitora e corrige cada uma independentemente |
| Cliente desliga e liga o PC | Printer Guardian inicia automaticamente |

### Troubleshooting

**Impressora continua offline após 30 segundos:**

- Verifique se o cabo USB está funcionando (teste com outro dispositivo)
- Verifique se o driver da impressora está instalado corretamente
- Abra `printer-guardian.log` e procure pela linha `USB Status:` para ver o status exato reportado pelo Windows
- Procure por `NOT_FIXED` no log para confirmar que o programa tentou corrigir

**Impressora Bluetooth não reconecta:**

- Verifique se a impressora está ligada e dentro do alcance Bluetooth
- O programa tenta reativar automaticamente, mas se a impressora estiver fisicamente fora de alcance, uma notificação de alerta será exibida
- Em último caso, remova e repareie o dispositivo Bluetooth manualmente via Configurações do Windows

**O programa não está rodando:**

- Verifique no Gerenciador de Tarefas se `PrinterGuardian.exe` aparece
- Se não aparece, execute o arquivo manualmente com duplo clique
- Verifique se o antivírus não bloqueou o executável

**Muitas notificações aparecendo:**

- Edite o `config.json` e aumente o `checkInterval` (ex: 30 ou 60 segundos)
- Ou identifique a impressora problemática e adicione na `blacklist`

---

## Licença

Projeto fornecido como-is para uso em ambientes de produção.
