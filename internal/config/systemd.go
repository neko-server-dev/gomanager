package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultServiceFileName = "gomanager.service"

func EnsureServiceFile(configPath string) error {
	if configPath == "" {
		configPath = DefaultPath
	}

	servicePath := filepath.Join(filepath.Dir(configPath), DefaultServiceFileName)
	if _, err := os.Stat(servicePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat service file: %w", err)
	}

	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("abs config path: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path: %w", err)
	}
	absExec, err := filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("abs executable path: %w", err)
	}

	content := renderService(absExec, absConfig)
	if err := os.WriteFile(servicePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}
	return nil
}

func renderService(execPath, configPath string) string {
	wd := filepath.Dir(configPath)

	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=gomanager IP blacklist API\n")
	b.WriteString("Documentation=https://github.com/neko-server-dev/gomanager\n")
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")

	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", wd)
	fmt.Fprintf(&b, "ExecStart=%q -config %q\n", execPath, configPath)
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n\n")

	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")

	return b.String()
}
