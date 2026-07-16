package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultCheckInterval = 15 * time.Second
	ConfigFileName       = "config.json"
	LogFileName          = "printer-guardian.log"
	BackupDirName        = "backups"
)

type Config struct {
	CheckInterval             int      `json:"checkInterval"`
	EnableUSBFix              bool     `json:"enableUSBFix"`
	EnableSNMPFix             bool     `json:"enableSNMPFix"`
	EnableBluetoothFix        bool     `json:"enableBluetoothFix"`
	EnableNewPrinterDetection bool     `json:"enableNewPrinterDetection"`
	EnableSelfMonitoring      bool     `json:"enableSelfMonitoring"`
	EnableQZTrayWatch         bool     `json:"enableQZTrayWatch"`
	Whitelist                 []string `json:"whitelist"`
	Blacklist                 []string `json:"blacklist"`
	MaintenanceMode           bool     `json:"maintenanceMode"`
}

type PrinterBackup struct {
	PrinterName string
	PortName    string
	Timestamp   time.Time
}

var (
	config         Config
	logger         *log.Logger
	logFile        *os.File
	backupDir      string
	configPath     string
	usbFixCooldown = map[string]time.Time{}
	btFixCooldown  = map[string]time.Time{}
	fixCooldownD   = 5 * time.Minute
)

func main() {
	initLogger()
	defer logFile.Close()
	logger.Println("=== Printer Guardian Iniciado ===")
	loadConfig()
	initBackupDir()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		logger.Println("Printer Guardian encerrando...")
		logFile.Close()
		os.Exit(0)
	}()

	go initTrayIcon()

	for {
		loadConfig()
		if !config.MaintenanceMode {
			logger.Println("Iniciando ciclo de verificação...")
			if config.EnableUSBFix {
				fixAllOfflineUSBPrinters()
			}
			if config.EnableSNMPFix {
				fixNetworkPrintersSNMP()
			}
			if config.EnableNewPrinterDetection {
				detectNewPrinters()
			}
			if config.EnableSelfMonitoring {
				selfHealthCheck()
			}
			if config.EnableBluetoothFix {
				fixBluetoothPrinters()
			}
			if config.EnableQZTrayWatch {
				watchQZTray()
			}
		} else {
			logger.Println("Modo manutenção ativo - pulando verificação")
		}
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
	}
}

func initLogger() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Erro ao obter caminho do executável:", err)
	}
	exeDir := filepath.Dir(exePath)
	logPath := filepath.Join(exeDir, LogFileName)
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Erro ao abrir arquivo de log:", err)
	}
	logger = log.New(logFile, "", log.LstdFlags)
}

func loadConfig() {
	config = Config{
		CheckInterval:             15,
		EnableUSBFix:              true,
		EnableSNMPFix:             true,
		EnableBluetoothFix:        true,
		EnableNewPrinterDetection: true,
		EnableSelfMonitoring:      true,
		EnableQZTrayWatch:         true,
		Whitelist:                 []string{},
		Blacklist:                 []string{},
		MaintenanceMode:           false,
	}
	exePath, err := os.Executable()
	if err != nil {
		logger.Println("Erro ao obter caminho do executável:", err)
		return
	}
	exeDir := filepath.Dir(exePath)
	configPath = filepath.Join(exeDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Println("Arquivo de configuração não encontrado, usando padrões")
		saveConfig()
		return
	}
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Println("Erro ao ler configuração, usando padrões:", err)
		return
	}
	logger.Printf("Configuração carregada: CheckInterval=%ds, USB=%v, SNMP=%v",
		config.CheckInterval, config.EnableUSBFix, config.EnableSNMPFix)
}

func saveConfig() {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logger.Println("Erro ao serializar configuração:", err)
		return
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.Println("Erro ao salvar configuração:", err)
		return
	}
	logger.Println("Configuração salva")
}

func initBackupDir() {
	exePath, err := os.Executable()
	if err != nil {
		logger.Println("Erro ao obter caminho do executável:", err)
		return
	}
	exeDir := filepath.Dir(exePath)
	backupDir = filepath.Join(exeDir, BackupDirName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.Println("Erro ao criar diretório de backup:", err)
		return
	}
	logger.Println("Diretório de backup inicializado:", backupDir)
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(`\`, "_", `/`, "_", `:`, "_", `*`, "_", `?`, "_", `"`, "_", `<`, "_", `>`, "_", `|`, "_")
	clean := replacer.Replace(name)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		clean = "unknown"
	}
	upper := strings.ToUpper(clean)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" {
		clean += "_printer"
	}
	if (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && len(upper) == 4 {
		last := upper[3]
		if last >= '1' && last <= '9' {
			clean += "_printer"
		}
	}
	return clean
}

func backupPrinterPort(printerName, portName string) error {
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.json", sanitizeFileName(printerName)))
	if _, err := os.Stat(backupFile); err == nil {
		return nil
	}
	backup := PrinterBackup{
		PrinterName: printerName,
		PortName:    portName,
		Timestamp:   time.Now(),
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return err
	}
	logger.Printf("Backup criado para impressora %s (porta %s)", printerName, portName)
	return nil
}

func restorePrinterPort(printerName string) error {
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.json", sanitizeFileName(printerName)))
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	var backup PrinterBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return err
	}
	psScript := fmt.Sprintf(`Set-Printer -Name "%s" -PortName "%s"`, printerName, backup.PortName)
	_, err = runPowerShell(psScript)
	if err != nil {
		return err
	}
	logger.Printf("Restore realizado para impressora %s (porta %s)", printerName, backup.PortName)
	return nil
}

func shouldProcessPrinter(printerName string) bool {
	for _, name := range config.Blacklist {
		if name == printerName {
			logger.Printf("Impressora %s na blacklist, ignorando", printerName)
			return false
		}
	}
	if len(config.Whitelist) == 0 {
		return true
	}
	for _, name := range config.Whitelist {
		if name == printerName {
			return true
		}
	}
	logger.Printf("Impressora %s não está na whitelist, ignorando", printerName)
	return false
}

func toPowerShellArray(items []string) string {
	if len(items) == 0 {
		return "@()"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''"))
	}
	return "@(" + strings.Join(parts, ", ") + ")"
}

func showNotification(title, message string) {
	psScript := fmt.Sprintf(`
		Add-Type -AssemblyName System.Windows.Forms
		$balloon = New-Object System.Windows.Forms.NotifyIcon
		$balloon.Icon = [System.Drawing.SystemIcons]::Information
		$balloon.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Info
		$balloon.BalloonTipTitle = "%s"
		$balloon.BalloonTipText = "%s"
		$balloon.Visible = $true
		$balloon.ShowBalloonTip(5000)
		Start-Sleep -Seconds 5
		$balloon.Dispose()
	`, title, message)
	runPowerShell(psScript)
	logger.Printf("Notificação: %s - %s", title, message)
}

func fixAllOfflineUSBPrinters() {
	psScript := fmt.Sprintf(`
		$okStatuses = @('Normal', 'Idle', 'ManualFeed', 'IoActive', 'Busy', 'Printing', 'Waiting', 'Processing', 'Initializing', 'WarmingUp', 'PowerSave')
		$whitelist = %s
		$blacklist = %s
		$allUsbPrinters = Get-Printer | Where-Object { $_.PortName -like "USB*" }
		$allowedPrinters = $allUsbPrinters | Where-Object {
			$name = $_.Name
			($blacklist -notcontains $name) -and (($whitelist.Count -eq 0) -or ($whitelist -contains $name))
		}
		if (-not $allowedPrinters) {
			Write-Output "USB_NO_PRINTERS"
			exit
		}
		foreach ($p in $allowedPrinters) {
			Write-Output "USB_STATUS|$($p.Name)|$($p.PortName)|$($p.PrinterStatus)"
		}
		$offlinePrinters = $allowedPrinters | Where-Object {
			$s = $_.PrinterStatus
			($okStatuses -notcontains $s) -and ($s -ne 0)
		}
		if (-not $offlinePrinters) {
			Write-Output "USB_ALL_OK"
			exit
		}
		$allUsbPorts = Get-PrinterPort | Where-Object { $_.Name -like "USB*" } | Sort-Object Name -Descending
		if (-not $allUsbPorts) {
			Write-Output "USB_NO_PORTS"
			exit
		}
		$usedPorts = @{}
		foreach ($p in $allUsbPrinters) {
			$s = $p.PrinterStatus
			if ((($okStatuses -contains $s) -or ($s -eq 0)) -and -not $usedPorts.ContainsKey($p.PortName)) {
				$usedPorts[$p.PortName] = $p.Name
			}
		}
		foreach ($printer in $offlinePrinters) {
			$printerName = $printer.Name
			$oldPort = $printer.PortName
			$fixed = $false
			foreach ($port in $allUsbPorts) {
				if ($port.Name -eq $oldPort) { continue }
				if ($usedPorts.ContainsKey($port.Name) -and $usedPorts[$port.Name] -ne $printerName) { continue }
				Set-Printer -Name $printerName -PortName $port.Name
				Start-Sleep -Seconds 2
				$check = Get-Printer -Name $printerName
				$s = $check.PrinterStatus
				if (($okStatuses -contains $s) -or ($s -eq 0)) {
					Write-Output "FIXED|$printerName|$oldPort|$($port.Name)"
					$usedPorts[$port.Name] = $printerName
					$fixed = $true
					break
				}
				Set-Printer -Name $printerName -PortName $oldPort
			}
			if (-not $fixed) {
				Write-Output "NOT_FIXED|$printerName|$oldPort"
			}
		}
	`, toPowerShellArray(config.Whitelist), toPowerShellArray(config.Blacklist))
	out, err := runPowerShell(psScript)
	if err != nil {
		logger.Println("Erro ao corrigir impressoras USB:", err)
		return
	}
	lines := parsePowerShellOutput(out)
	for _, line := range lines {
		if len(line) < 1 {
			continue
		}
		switch line[0] {
		case "USB_STATUS":
			if len(line) >= 4 {
				logger.Printf("USB Status: %s | porta=%s | status=%s", line[1], line[2], line[3])
				if err := backupPrinterPort(line[1], line[2]); err != nil {
					logger.Printf("Erro ao fazer backup da porta %s para %s: %v", line[2], line[1], err)
				}
			}
		case "USB_ALL_OK":
			logger.Println("USB Fix: todas as impressoras USB estão OK")
		case "USB_NO_PRINTERS":
			logger.Println("USB Fix: nenhuma impressora USB encontrada")
		case "USB_NO_PORTS":
			logger.Println("USB Fix: impressoras offline detectadas mas nenhuma porta USB disponível para tentar")
			showNotification("Impressora Offline", "Impressora(s) offline detectada(s), mas nenhuma porta USB está disponível")
		case "FIXED":
			if len(line) >= 4 {
				printerName, oldPort, newPort := line[1], line[2], line[3]
				delete(usbFixCooldown, printerName)
				if shouldProcessPrinter(printerName) {
					logger.Printf("USB Fix: %s corrigida — de %s para %s", printerName, oldPort, newPort)
					showNotification("Impressora Corrigida", fmt.Sprintf("%s movida para porta %s", printerName, newPort))
				}
			}
		case "NOT_FIXED":
			if len(line) >= 3 {
				name, port := line[1], line[2]
				if last, ok := usbFixCooldown[name]; ok && time.Since(last) < fixCooldownD {
					logger.Printf("USB Fix: %s em cooldown, próxima tentativa em %.0f min", name, fixCooldownD.Minutes()-time.Since(last).Minutes())
					break
				}
				usbFixCooldown[name] = time.Now()
				logger.Printf("USB Fix: %s não foi corrigida (porta atual: %s) — impressora pode estar desligada", name, port)
				if shouldProcessPrinter(name) {
					showNotification("Impressora Offline", fmt.Sprintf("%s está offline — verifique se está ligada (porta %s)", name, port))
				}
			}
		}
	}
}

func fixNetworkPrintersSNMP() {
	psScript := `
		$tcpPorts = Get-PrinterPort | Where-Object { 
			($_.Name -match "^IP_" -or $_.Name -match "^[0-9]{1,3}\.") -and $_.SNMPEnabled -eq $true
		}
		$fixedPorts = @()
		foreach ($port in $tcpPorts) {
			Set-PrinterPort -Name $port.Name -SNMPEnabled $false
			$fixedPorts += $port.Name
		}
		if ($fixedPorts.Count -gt 0) {
			Restart-Service -Name Spooler -Force
			Write-Output "SNMP_FIXED|$($fixedPorts -join ',')"
		}
	`
	out, err := runPowerShell(psScript)
	if err != nil {
		logger.Println("Erro ao corrigir SNMP:", err)
		return
	}
	lines := parsePowerShellOutput(out)
	for _, line := range lines {
		if len(line) >= 2 && line[0] == "SNMP_FIXED" {
			ports := line[1]
			logger.Printf("SNMP desativado nas portas: %s", ports)
			showNotification("SNMP Desativado", fmt.Sprintf("Portas corrigidas: %s", ports))
		}
	}
}

func fixBluetoothPrinters() {
	psScript := fmt.Sprintf(`
		$okStatuses = @('Normal', 'Idle', 'ManualFeed', 'IoActive', 'Busy', 'Printing', 'Waiting', 'Processing', 'Initializing', 'WarmingUp', 'PowerSave')
		$whitelist = %s
		$blacklist = %s
		$btPrinters = Get-Printer | Where-Object {
			$_.PortName -like "COM*" -or $_.PortName -like "BTHPRINT*"
		}
		$allowedBtPrinters = $btPrinters | Where-Object {
			$name = $_.Name
			($blacklist -notcontains $name) -and (($whitelist.Count -eq 0) -or ($whitelist -contains $name))
		}
		if (-not $allowedBtPrinters) {
			Write-Output "BT_NO_PRINTERS"
			exit
		}
		foreach ($p in $allowedBtPrinters) {
			Write-Output "BT_STATUS|$($p.Name)|$($p.PortName)|$($p.PrinterStatus)"
		}
		$offlineBt = $allowedBtPrinters | Where-Object {
			$s = $_.PrinterStatus
			($okStatuses -notcontains $s) -and ($s -ne 0)
		}
		if (-not $offlineBt) {
			Write-Output "BT_ALL_OK"
			exit
		}
		foreach ($printer in $offlineBt) {
			$printerName = $printer.Name
			$portName    = $printer.PortName
			$recovered   = $false

			$devices = Get-PnpDevice | Where-Object {
				($_.Class -eq 'Ports' -or $_.Class -eq 'Bluetooth' -or $_.Class -eq 'PrintQueue') -and
				$_.FriendlyName -like "*$printerName*"
			}
			foreach ($dev in $devices) {
				try {
					Enable-PnpDevice -InstanceId $dev.InstanceId -Confirm:$false -ErrorAction Stop
					Start-Sleep -Seconds 3
					$check = Get-Printer -Name $printerName
					$s = $check.PrinterStatus
					if (($okStatuses -contains $s) -or ($s -eq 0)) {
						Write-Output "BT_FIXED|$printerName|$portName"
						$recovered = $true
						break
					}
				} catch {}
			}

			if (-not $recovered) {
				try {
					Restart-Service -Name Spooler -Force -ErrorAction Stop
					Start-Sleep -Seconds 3
					$check = Get-Printer -Name $printerName
					$s = $check.PrinterStatus
					if (($okStatuses -contains $s) -or ($s -eq 0)) {
						Write-Output "BT_FIXED_SPOOLER|$printerName|$portName"
						$recovered = $true
					}
				} catch {}
			}

			if (-not $recovered) {
				Write-Output "BT_NOT_FIXED|$printerName|$portName"
			}
		}
	`,
		toPowerShellArray(config.Whitelist),
		toPowerShellArray(config.Blacklist))
	out, err := runPowerShell(psScript)
	if err != nil {
		logger.Println("Erro ao verificar impressoras Bluetooth:", err)
		return
	}
	lines := parsePowerShellOutput(out)
	for _, line := range lines {
		if len(line) < 1 {
			continue
		}
		switch line[0] {
		case "BT_NO_PRINTERS":
			logger.Println("Bluetooth Fix: nenhuma impressora Bluetooth encontrada")
		case "BT_ALL_OK":
			logger.Println("Bluetooth Fix: todas as impressoras Bluetooth estão OK")
		case "BT_STATUS":
			if len(line) >= 4 {
				logger.Printf("Bluetooth Status: %s | porta=%s | status=%s", line[1], line[2], line[3])
			}
		case "BT_FIXED":
			if len(line) >= 3 {
				name, port := line[1], line[2]
				if shouldProcessPrinter(name) {
					logger.Printf("Bluetooth Fix: %s recuperada via PnP (porta %s)", name, port)
					showNotification("Impressora Bluetooth Recuperada", fmt.Sprintf("%s reconectada automaticamente", name))
				}
			}
		case "BT_FIXED_SPOOLER":
			if len(line) >= 3 {
				name, port := line[1], line[2]
				if shouldProcessPrinter(name) {
					logger.Printf("Bluetooth Fix: %s recuperada via reinício do Spooler (porta %s)", name, port)
					showNotification("Impressora Bluetooth Recuperada", fmt.Sprintf("%s reconectada após reinício do Spooler", name))
				}
			}
		case "BT_NOT_FIXED":
			if len(line) >= 3 {
				name, port := line[1], line[2]
				if last, ok := btFixCooldown[name]; ok && time.Since(last) < fixCooldownD {
					logger.Printf("Bluetooth Fix: %s em cooldown, próxima tentativa em %.0f min", name, fixCooldownD.Minutes()-time.Since(last).Minutes())
					break
				}
				btFixCooldown[name] = time.Now()
				logger.Printf("Bluetooth Fix: %s não recuperada (porta %s) — impressora pode estar fora de alcance", name, port)
				if shouldProcessPrinter(name) {
					showNotification("Impressora Bluetooth Offline", fmt.Sprintf("%s não responde — verifique se está ligada e no alcance", name))
				}
			}
		}
	}
}

func detectNewPrinters() {
	psScript := `
		$allPrinters = Get-Printer | Select-Object Name, PortName, DriverName, CreatedDate
		foreach ($printer in $allPrinters) {
			if ($printer.CreatedDate -and (Get-Date).AddMinutes(-30) -lt $printer.CreatedDate) {
				Write-Output "NEW_PRINTER|$($printer.Name)|$($printer.PortName)"
			}
		}
	`
	out, err := runPowerShell(psScript)
	if err != nil {
		logger.Println("Erro ao detectar novas impressoras:", err)
		return
	}
	lines := parsePowerShellOutput(out)
	for _, line := range lines {
		if len(line) >= 3 && line[0] == "NEW_PRINTER" {
			printerName := line[1]
			portName := line[2]
			logger.Printf("Nova impressora detectada: %s na porta %s", printerName, portName)
			showNotification("Nova Impressora", fmt.Sprintf("%s detectada em %s", printerName, portName))
			if shouldProcessPrinter(printerName) {
				applyDefaultSettings(printerName)
			}
		}
	}
}

func applyDefaultSettings(printerName string) {
	logger.Printf("Aplicando configurações padrão para %s", printerName)
}

func selfHealthCheck() {
	if logFile == nil {
		logger.Println("ALERTA: Arquivo de log não está aberto")
		return
	}
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		logger.Println("ALERTA: Diretório de backup não existe, recriando...")
		initBackupDir()
	}
	logger.Println("Self-health check OK")
}

func initTrayIcon() {
	logger.Println("Tray icon inicializado (implementação básica)")
}

func watchQZTray() {
	psScript := `
		$excluded = @('powershell', 'pwsh', 'cmd', 'conhost')
		$proc = Get-Process | Where-Object {
			($excluded -notcontains $_.ProcessName) -and
			(
				$_.ProcessName -like '*qz*' -or
				($_.Path -and $_.Path -like '*QZ Tray*') -or
				($_.MainWindowTitle -and $_.MainWindowTitle -like '*QZ Tray*')
			)
		}
		if (-not $proc) {
			$proc = Get-Process | Where-Object {
				($_.ProcessName -like 'java*' -or $_.ProcessName -like 'javaw*') -and
				(
					($_.Path -and $_.Path -like '*QZ Tray*') -or
					($_.MainWindowTitle -and $_.MainWindowTitle -like '*QZ Tray*')
				)
			} | Select-Object -First 1
		}
		if (-not $proc) {
			$proc = Get-CimInstance Win32_Process | Where-Object {
				$baseName = [System.IO.Path]::GetFileNameWithoutExtension($_.Name)
				($excluded -notcontains $baseName) -and
				(
					($_.ExecutablePath -and $_.ExecutablePath -like '*QZ Tray*') -or
					($_.CommandLine -like '*qz-tray.jar*') -or
					($_.CommandLine -like '*qz-tray.exe*')
				)
			} | Select-Object -First 1
		}
		if ($proc) {
			$found = $proc | Select-Object -First 1
			Write-Output "QZ_RUNNING|$($found.ProcessName)|$($found.Path)"
		} else {
			$paths = @(
				"$env:ProgramFiles\QZ Tray\qz-tray.exe",
				"${env:ProgramFiles(x86)}\QZ Tray\qz-tray.exe",
				"$env:USERPROFILE\AppData\Local\Programs\QZ Tray\qz-tray.exe",
				"$env:USERPROFILE\Desktop\QZ Tray\qz-tray.exe"
			)
			foreach ($path in $paths) {
				if (Test-Path $path) {
					Start-Process -FilePath $path -WindowStyle Hidden
					Write-Output "QZ_STARTED|$path"
					return
				}
			}
			Write-Output "QZ_NOT_FOUND"
		}
	`
	out, err := runPowerShell(psScript)
	logger.Printf("QZ Tray raw output: %q", out)
	if err != nil {
		logger.Println("Erro ao verificar QZ Tray:", err)
		return
	}
	lines := parsePowerShellOutput(out)
	for _, line := range lines {
		if len(line) >= 1 {
			switch line[0] {
			case "QZ_RUNNING":
				name, path := "", ""
				if len(line) >= 2 {
					name = line[1]
				}
				if len(line) >= 3 {
					path = line[2]
				}
				logger.Printf("QZ Tray já está rodando: %s (%s)", name, path)
			case "QZ_STARTED":
				path := ""
				if len(line) >= 2 {
					path = line[1]
				}
				logger.Printf("QZ Tray reiniciado: %s", path)
				showNotification("QZ Tray Reiniciado", "O QZ Tray foi reaberto automaticamente")
			case "QZ_NOT_FOUND":
				logger.Println("ALERTA: QZ Tray não encontrado em nenhum caminho padrão")
			}
		}
	}
}

func parsePowerShellOutput(output string) [][]string {
	lines := []string{}
	for _, line := range bytes.Split([]byte(output), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			lines = append(lines, string(trimmed))
		}
	}
	result := [][]string{}
	for _, line := range lines {
		fields := []string{}
		for _, field := range bytes.Split([]byte(line), []byte("|")) {
			fields = append(fields, string(bytes.TrimSpace(field)))
		}
		if len(fields) > 0 {
			result = append(result, fields)
		}
	}
	return result
}

func runPowerShell(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		logger.Println("PowerShell timeout: processo encerrado após 60s")
	}
	if stderr.Len() > 0 {
		logger.Printf("PowerShell stderr: %s", stderr.String())
	}
	return out.String(), err
}
