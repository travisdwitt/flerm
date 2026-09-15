package config

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
	Resume        bool
}

func Load() *Config {
	config := &Config{
		SaveDirectory: "",
		StartMenu:     true,
		Confirmations: true,
		Resume:        true,
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
		case "resume", "resumelast", "resume_last":
			config.Resume = strings.ToLower(value) == "true"
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

// lastFileRecord holds the path of the most recently opened chart, so the start
// menu can offer to resume it on the next run. Empty when resume is off.
func (c *Config) lastFileRecord() string {
	if !c.Resume {
		return ""
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".flerm_last")
}

func (c *Config) RememberLastFile(path string) {
	record := c.lastFileRecord()
	if record == "" || path == "" {
		return
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	os.WriteFile(record, []byte(path+"\n"), 0644)
}

// LastFile returns the remembered chart, or "" if resume is off, nothing was
// opened yet, or the file has since been moved or deleted.
func (c *Config) LastFile() string {
	record := c.lastFileRecord()
	if record == "" {
		return ""
	}
	data, err := os.ReadFile(record)
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(data))
	if path == "" {
		return ""
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return ""
	}
	return path
}
