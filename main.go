package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		initRepo(args)
	case "clone":
		cloneRepo(args)
	case "add":
		addFiles(args)
	case "commit":
		commitChanges(args)
	case "push":
		pushChanges(args)
	case "pull":
		pullChanges(args)
	case "status":
		showStatus(args)
	case "branch":
		handleBranch(args)
	case "checkout":
		checkoutBranch(args)
	case "log":
		showLog(args)
	case "config":
		handleConfig(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("mgit - A go-git wrapper")
	fmt.Println("Usage: mgit <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  init            Initialize a new repository")
	fmt.Println("  clone <url>     Clone a repository")
	fmt.Println("  add <files...>  Add files to staging")
	fmt.Println("  commit -m <msg> Commit staged changes")
	fmt.Println("  push            Push commits to remote")
	fmt.Println("  pull            Pull changes from remote")
	fmt.Println("  status          Show repository status")
	fmt.Println("  branch          List branches")
	fmt.Println("  branch <name>   Create a new branch")
	fmt.Println("  checkout <ref>  Checkout a branch or commit")
	fmt.Println("  log             Show commit history")
	fmt.Println("  config          Get and set configuration values")
}

func initRepo(args []string) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	_, err := git.PlainInit(path, false)
	if err != nil {
		fmt.Printf("Error initializing repository: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Initialized empty Git repository in %s\n", path)
}

func cloneRepo(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit clone <url> [path]")
		os.Exit(1)
	}

	url := args[0]
	path := "."
	if len(args) > 1 {
		path = args[1]
	}

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Printf("Error cloning repository: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cloned repository %s to %s\n", url, path)
}

func getRepo() *git.Repository {
	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("Error opening repository: %s\n", err)
		os.Exit(1)
	}
	return repo
}

func addFiles(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit add <files...>")
		os.Exit(1)
	}

	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	for _, file := range args {
		_, err := w.Add(file)
		if err != nil {
			fmt.Printf("Error adding file %s: %s\n", file, err)
			os.Exit(1)
		}
	}
	fmt.Println("Changes staged for commit")
}

func commitChanges(args []string) {
	message := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-m" && i+1 < len(args) {
			message = args[i+1]
			break
		}
	}

	if message == "" {
		fmt.Println("Usage: mgit commit -m <message>")
		os.Exit(1)
	}

	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	commit, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  GetConfigValue("user.name", "mgit User"),
			Email: GetConfigValue("user.email", "mgit@example.com"),
			When:  time.Now(),
		},
	})
	if err != nil {
		fmt.Printf("Error committing changes: %s\n", err)
		os.Exit(1)
	}

	obj, err := repo.CommitObject(commit)
	if err != nil {
		fmt.Printf("Error getting commit: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Committed changes [%s]: %s\n", obj.Hash.String()[:7], message)
}

/* 
	config related changes below
*/

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
	
	return ioutil.WriteFile(file, []byte(content), 0644)
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

// handleConfig handles the config command
func handleConfig(args []string) {
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

func pushChanges(args []string) {
	repo := getRepo()
	
	// Get authentication if provided through environment variables
	auth := &http.BasicAuth{
		Username: os.Getenv("MGIT_USERNAME"),
		Password: os.Getenv("MGIT_PASSWORD"),
	}

	// Don't use auth if credentials aren't provided
	var authOption *http.BasicAuth
	if auth.Username != "" && auth.Password != "" {
		authOption = auth
	}

	err := repo.Push(&git.PushOptions{
		Auth:     authOption,
		Progress: os.Stdout,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			fmt.Println("Everything up-to-date")
			return
		}
		fmt.Printf("Error pushing changes: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Changes pushed to remote")
}

func pullChanges(args []string) {
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	err = w.Pull(&git.PullOptions{
		Progress: os.Stdout,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			fmt.Println("Already up-to-date")
			return
		}
		fmt.Printf("Error pulling changes: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Changes pulled from remote")
}

func showStatus(args []string) {
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	status, err := w.Status()
	if err != nil {
		fmt.Printf("Error getting status: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Current branch:", getCurrentBranch(repo))
	fmt.Println()
	
	if status.IsClean() {
		fmt.Println("Nothing to commit, working tree clean")
		return
	}

	fmt.Println("Changes to be committed:")
	for file, fileStatus := range status {
		if fileStatus.Staging == git.Added {
			fmt.Printf("  new file:   %s\n", file)
		} else if fileStatus.Staging == git.Modified {
			fmt.Printf("  modified:   %s\n", file)
		} else if fileStatus.Staging == git.Deleted {
			fmt.Printf("  deleted:    %s\n", file)
		}
	}
	fmt.Println()

	fmt.Println("Changes not staged for commit:")
	for file, fileStatus := range status {
		if fileStatus.Worktree == git.Modified {
			fmt.Printf("  modified:   %s\n", file)
		} else if fileStatus.Worktree == git.Deleted {
			fmt.Printf("  deleted:    %s\n", file)
		}
	}
	fmt.Println()

	fmt.Println("Untracked files:")
	for file, fileStatus := range status {
		if fileStatus.Worktree == git.Untracked {
			fmt.Printf("  %s\n", file)
		}
	}
}

func getCurrentBranch(repo *git.Repository) string {
	head, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD: %s\n", err)
		return "unknown"
	}
	
	if head.Name().IsBranch() {
		return head.Name().Short()
	}
	
	return head.Hash().String()[:7]
}

func handleBranch(args []string) {
	repo := getRepo()
	
	if len(args) == 0 {
		// List branches
		branches, err := repo.Branches()
		if err != nil {
			fmt.Printf("Error listing branches: %s\n", err)
			os.Exit(1)
		}
		
		currentBranch := getCurrentBranch(repo)
		fmt.Println("Branches:")
		
		err = branches.ForEach(func(branch *plumbing.Reference) error {
			name := branch.Name().Short()
			if name == currentBranch {
				fmt.Printf("* %s\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error iterating branches: %s\n", err)
			os.Exit(1)
		}
	} else {
		// Create a new branch
		branchName := args[0]
		
		w, err := repo.Worktree()
		if err != nil {
			fmt.Printf("Error getting worktree: %s\n", err)
			os.Exit(1)
		}
		
		head, err := repo.Head()
		if err != nil {
			fmt.Printf("Error getting HEAD: %s\n", err)
			os.Exit(1)
		}
		
		err = w.Checkout(&git.CheckoutOptions{
			Hash:   head.Hash(),
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: true,
		})
		if err != nil {
			fmt.Printf("Error creating branch %s: %s\n", branchName, err)
			os.Exit(1)
		}
		
		fmt.Printf("Switched to a new branch '%s'\n", branchName)
	}
}

func checkoutBranch(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit checkout <branch>")
		os.Exit(1)
	}
	
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}
	
	branchName := args[0]
	
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		// Maybe it's a commit hash?
		hash := plumbing.NewHash(branchName)
		err = w.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err != nil {
			fmt.Printf("Error checking out %s: %s\n", branchName, err)
			os.Exit(1)
		}
		fmt.Printf("Checked out commit %s\n", branchName)
	} else {
		fmt.Printf("Switched to branch '%s'\n", branchName)
	}
}

func showLog(args []string) {
	repo := getRepo()
	
	// Get the HEAD reference
	ref, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD: %s\n", err)
		os.Exit(1)
	}
	
	// Get commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		fmt.Printf("Error getting commit: %s\n", err)
		os.Exit(1)
	}
	
	// Get commit history
	commitIter, err := repo.Log(&git.LogOptions{From: commit.Hash})
	if err != nil {
		fmt.Printf("Error getting log: %s\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Commit History:")
	err = commitIter.ForEach(func(c *object.Commit) error {
		fmt.Printf("Commit: %s\n", c.Hash.String())
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		fmt.Printf("Date:   %s\n", c.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700"))
		fmt.Printf("\n    %s\n\n", c.Message)
		return nil
	})
	if err != nil {
		fmt.Printf("Error iterating commits: %s\n", err)
		os.Exit(1)
	}
}