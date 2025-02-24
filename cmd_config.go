package main

import (
	"fmt"
	"os"
)

// HandleConfig handles the config command
func HandleConfig(args []string) {
	if len(args) == 0 {
		// List all config values
		listConfig()
		return
	}

	// Check for --global flag
	isGlobal := false
	filteredArgs := []string{}
	for _, arg := range args {
		if arg == "--global" {
			isGlobal = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) == 1 {
		// Get a config value
		value := GetConfigValue(args[0], "")
		if value == "" {
			fmt.Printf("No value set for %s\n", args[0])
		} else {
			fmt.Println(value)
		}
		return
	}

	if len(args) == 2 {
		// Set a config value
		key := args[0]
		value := args[1]
		err := SetConfigValue(key, value, isGlobal)
		if err != nil {
			fmt.Printf("Error setting config value: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s to %s in %s config\n", key, value, getConfigType(isGlobal))
		return
	}

	fmt.Println("Usage: mgit config [--global] [<key> [<value>]]")
	os.Exit(1)
}

// listConfig lists all config values
func listConfig() {
	// List local config
	localConfigPath := GetConfigFilePath(false)
	localConfig, err := LoadConfig(localConfigPath)
	if err == nil && len(localConfig.Sections) > 0 {
		fmt.Println("Local config:")
		printConfig(localConfig)
		fmt.Println()
	}

	// List global config
	globalConfigPath := GetConfigFilePath(true)
	globalConfig, err := LoadConfig(globalConfigPath)
	if err == nil && len(globalConfig.Sections) > 0 {
		fmt.Println("Global config:")
		printConfig(globalConfig)
	}
}

// printConfig prints a config
func printConfig(config *Config) {
	for section, values := range config.Sections {
		for key, value := range values {
			fmt.Printf("\t%s.%s=%s\n", section, key, value)
		}
	}
}

// getConfigType returns the type of config
func getConfigType(global bool) string {
	if global {
		return "global"
	}
	return "local"
}