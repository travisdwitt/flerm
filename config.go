package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	SaveDirectory      string
	StartMenu          bool
	Confirmations      bool
}

func loadConfig() *Config {
	config := &Config{
		SaveDirectory: "", // Empty means use current directory
		StartMenu:     true,
		Confirmations: true,
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config // Return defaults if we can't get home directory
	}

	// Check for .flermrc file
	configPath := filepath.Join(homeDir, ".flermrc")
	file, err := os.Open(configPath)
	if err != nil {
		return config // Return defaults if file doesn't exist
	}
	defer file.Close()

	// Parse config file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "savedirectory", "save_directory", "savedir":
			// Expand ~ to home directory if present
			if strings.HasPrefix(value, "~") {
				value = filepath.Join(homeDir, strings.TrimPrefix(value, "~"))
			}
			// Expand to absolute path
			if !filepath.IsAbs(value) {
				absPath, err := filepath.Abs(value)
				if err == nil {
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
	
	// Ensure directory exists
	os.MkdirAll(c.SaveDirectory, 0755)
	
	return filepath.Join(c.SaveDirectory, filename)
}

