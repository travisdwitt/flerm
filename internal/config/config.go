package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	SaveDirectory string
	StartMenu     bool
	Confirmations bool
	Particles     bool
	Mouse         bool
	Resume        bool
}

const configName = ".flermrc"

func Defaults() *Config {
	return &Config{
		SaveDirectory: "",
		StartMenu:     true,
		Confirmations: true,
		Particles:     true,
		Mouse:         true,
		Resume:        true,
	}
}

func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configName), nil
}

func (c *Config) Save(path string) error {
	settings := fmt.Sprintf("savedirectory=%s\nstartmenu=%v\nconfirmations=%v\nresume=%v\nparticles=%v\nmouse=%v\n",
		c.SaveDirectory, c.StartMenu, c.Confirmations, c.Resume, c.Particles, c.Mouse)
	return os.WriteFile(path, []byte(settings), 0644)
}

func Load() *Config {
	config := Defaults()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config
	}

	configPath := filepath.Join(homeDir, configName)
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
			if value == "" {
				config.SaveDirectory = ""
				continue
			}
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
			config.StartMenu = isTrue(value)
		case "confirmations", "confirm":
			config.Confirmations = isTrue(value)
		case "particles", "animations":
			config.Particles = isTrue(value)
		case "disableparticles", "disable_particles", "disableanimations":
			config.Particles = !isTrue(value)
		case "resume":
			config.Resume = isTrue(value)
		case "mouse":
			config.Mouse = isTrue(value)
		case "disablemouse", "disable_mouse":
			config.Mouse = !isTrue(value)
		}
	}

	return config
}

func isTrue(value string) bool { return strings.ToLower(value) == "true" }

func (c *Config) GetSavePath(filename string) string {
	if c.SaveDirectory == "" {
		return filename
	}
	os.MkdirAll(c.SaveDirectory, 0755)
	return filepath.Join(c.SaveDirectory, filename)
}

func (c *Config) lastFileRecord() string {
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

func (c *Config) LastFile() string {
	if !c.Resume {
		return ""
	}
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
