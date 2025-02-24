package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents a git-like config file
type Config struct {
	Sections map[string]map[string]string
}

// Load config from file
func LoadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{
				Sections: make(map[string]map[string]string),
			}, nil
		}
		return nil, err
	}

	return parseConfig(string(data))
}

// Parse a config file content
func parseConfig(content string) (*Config, error) {
	config := &Config{
		Sections: make(map[string]map[string]string),
	}

	lines := strings.Split(content, "\n")
	currentSection := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue // Skip empty lines and comments
		}

		// Section header [section] or [section "subsection"]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := line[1 : len(line)-1]
			currentSection = sectionName
			if _, exists := config.Sections[currentSection]; !exists {
				config.Sections[currentSection] = make(map[string]string)
			}
			continue
		}

		if currentSection == "" {
			continue // No section defined yet
		}

		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Invalid format
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		config.Sections[currentSection][key] = value
	}

	return config, nil
}

// Save config to file
func (c *Config) Save(file string) error {
	content := ""
	
	for section, values := range c.Sections {
		if len(values) == 0 {
			continue
		}
		
		content += fmt.Sprintf("[%s]\n", section)
		for key, value := range values {
			content += fmt.Sprintf("\t%s = %s\n", key, value)
		}
		content += "\n"
	}
	
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(file, []byte(content), 0644)
}

// Get a config value
func (c *Config) Get(section, key string) string {
	if values, exists := c.Sections[section]; exists {
		return values[key]
	}
	return ""
}

// Set a config value
func (c *Config) Set(section, key, value string) {
	if _, exists := c.Sections[section]; !exists {
		c.Sections[section] = make(map[string]string)
	}
	c.Sections[section][key] = value
}

// GetConfigFilePath returns the path to the config file
func GetConfigFilePath(global bool) string {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, ".mgitconfig")
	}
	
	// Local config
	return ".mgit/config"
}

// GetConfigValue gets a config value from either local or global config
func GetConfigValue(key, defaultValue string) string {
	// First check environment variables (for backward compatibility)
	envKey := "MGIT_" + strings.ToUpper(strings.Replace(key, ".", "_", -1))
	if value, exists := os.LookupEnv(envKey); exists {
		return value
	}
	
	// Parse the key into section and name
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return defaultValue
	}
	
	section := parts[0]
	name := parts[1]
	
	// Check local config first
	localConfigPath := GetConfigFilePath(false)
	localConfig, err := LoadConfig(localConfigPath)
	if err == nil {
		value := localConfig.Get(section, name)
		if value != "" {
			return value
		}
	}
	
	// Then check global config
	globalConfigPath := GetConfigFilePath(true)
	globalConfig, err := LoadConfig(globalConfigPath)
	if err == nil {
		value := globalConfig.Get(section, name)
		if value != "" {
			return value
		}
	}
	
	return defaultValue
}

// SetConfigValue sets a config value in either local or global config
func SetConfigValue(key, value string, global bool) error {
	// Parse the key into section and name
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid config key format: %s", key)
	}
	
	section := parts[0]
	name := parts[1]
	
	configPath := GetConfigFilePath(global)
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	
	config.Set(section, name, value)
	return config.Save(configPath)
}