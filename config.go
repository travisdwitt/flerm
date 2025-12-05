package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	SaveDirectory string
	StartMenu     bool
	Confirmations bool
}

func loadConfig() *Config {
	config := &Config{
		SaveDirectory: "",
		StartMenu:     true,
		Confirmations: true,
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config
	}

	configPath := filepath.Join(homeDir, ".flermrc")
	file, err := os.Open(configPath)
	if err != nil {
		return config
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "savedirectory", "save_directory", "savedir":
			if strings.HasPrefix(value, "~") {
				value = filepath.Join(homeDir, strings.TrimPrefix(value, "~"))
			}
			if !filepath.IsAbs(value) {
				if absPath, err := filepath.Abs(value); err == nil {
					value = absPath
				}
			}
			config.SaveDirectory = value
		case "startmenu", "start_menu":
			config.StartMenu = strings.ToLower(value) == "true"
		case "confirmations", "confirm":
			config.Confirmations = strings.ToLower(value) == "true"
		}
	}

	return config
}

func (c *Config) GetSavePath(filename string) string {
	if c.SaveDirectory == "" {
		return filename
	}
	os.MkdirAll(c.SaveDirectory, 0755)
	return filepath.Join(c.SaveDirectory, filename)
}

