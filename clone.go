package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AuthToken represents an authentication token for a repository
type AuthToken struct {
	Token   string `json:"token"`
	RepoURL string `json:"repoUrl"`
	Access  string `json:"access"`
}

// TokenStore represents the token storage in mgitconfig
type TokenStore struct {
	Tokens []AuthToken `json:"tokens"`
}

// CloneOptions represents options for the clone command
type CloneOptions struct {
	NoCheckout bool
	Depth      int
	Branch     string
}

// HandleClone handles the clone command
func HandleClone(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit clone <url> [destination]")
		os.Exit(1)
	}

	url := args[0]
	destination := ""
	if len(args) > 1 {
		destination = args[1]
	} else {
		// If no destination is specified, use the last part of the URL as the directory name
		parts := strings.Split(url, "/")
		destination = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	// Normalize URL to ensure it doesn't end with a slash
	url = strings.TrimSuffix(url, "/")

	// Get token for the repository
	token := getTokenForRepo(url)

	// Clone the repository
	err := cloneRepository(url, destination, token)
	if err != nil {
		fmt.Printf("Error cloning repository: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully cloned repository to %s\n", destination)
}

// getTokenForRepo retrieves the authentication token for a repository URL
func getTokenForRepo(repoURL string) string {
	// Get the path to the mgit config file
	configPath := getTokenConfigPath()

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No authentication token found. Please authenticate first using the web interface.")
		os.Exit(1)
	}

	// Read the token file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading token file: %s\n", err)
		os.Exit(1)
	}

	// Parse the token store
	var store TokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		fmt.Printf("Error parsing token file: %s\n", err)
		os.Exit(1)
	}

	// Find the token for the repository
	for _, t := range store.Tokens {
    // Add diagnostic print statement
    fmt.Printf("Comparing URLs - Stored: %s, Current: %s\n", t.RepoURL, repoURL)
    
    // Check if the repo URL matches
    if matchRepoURL(t.RepoURL, repoURL) {
        fmt.Printf("Found matching token for %s\n", repoURL)
        return t.Token
    }
}

	fmt.Println("No authentication token found for this repository. Please authenticate first using the web interface.")
	os.Exit(1)
	return ""
}

// matchRepoURL checks if two repository URLs refer to the same repository
func matchRepoURL(storedURL, providedURL string) bool {
	// Normalize URLs by removing trailing slashes and .git suffix
	storedURL = strings.TrimSuffix(strings.TrimSuffix(storedURL, "/"), ".git")
	providedURL = strings.TrimSuffix(strings.TrimSuffix(providedURL, "/"), ".git")
	
	fmt.Printf("Matching URLs - Stored: %s, Provided: %s\n", storedURL, providedURL)
	
	// Check for exact match first
	if storedURL == providedURL {
			return true
	}
	
	// Extract the repository ID from both URLs
	storedRepoID := extractRepoIDFromAnyURL(storedURL)
	providedRepoID := extractRepoIDFromAnyURL(providedURL)
	
	fmt.Printf("Extracted RepoIDs - Stored: %s, Provided: %s\n", storedRepoID, providedRepoID)
	
	// Consider it a match if we can extract valid repo IDs and they match
	return storedRepoID != "" && providedRepoID != "" && storedRepoID == providedRepoID
}

// extractRepoIDFromAnyURL extracts the repository ID from any URL format
func extractRepoIDFromAnyURL(url string) string {
	// Handle API format: http://localhost:3003/api/mgit/repos/hello-world
	if strings.Contains(url, "/api/mgit/repos/") {
			parts := strings.Split(url, "/api/mgit/repos/")
			if len(parts) > 1 {
					return parts[1]
			}
	}
	
	// Handle direct format: http://localhost:3003/hello-world
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
			return parts[len(parts)-1]
	}
	
	return ""
}

// getTokenConfigPath returns the path to the token config file
func getTokenConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %s\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".mgitconfig", "tokens.json")
}

// cloneRepository clones a repository
func cloneRepository(url, destination, token string) error {
	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("error creating destination directory: %w", err)
	}

	// First, we use the mgit-fetch endpoint to get repository metadata
	// This requires authentication and will give us information about the repository
	fmt.Println("Fetching repository metadata...")
	repoInfo, err := fetchRepositoryInfo(url, token)
	if err != nil {
		return fmt.Errorf("error fetching repository metadata: %w", err)
	}

	fmt.Printf("Repository: %s\nAccess level: %s\n", repoInfo.Name, repoInfo.Access)

	// First, clone the Git data using git-upload-pack
	fmt.Println("Cloning Git repository data...")
	if err := cloneGitData(url, destination, token); err != nil {
		return fmt.Errorf("error cloning Git data: %w", err)
	}

	// Then, fetch and set up the MGit metadata
	fmt.Println("Fetching MGit metadata...")
	if err := fetchMGitMetadata(url, destination, token); err != nil {
		return fmt.Errorf("error fetching MGit metadata: %w", err)
	}

	// Set up the MGit configuration for the cloned repository
	if err := setupMGitConfig(destination, repoInfo); err != nil {
		return fmt.Errorf("error setting up MGit configuration: %w", err)
	}

	return nil
}
// RepositoryInfo represents repository information returned from the server
type RepositoryInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Access           string `json:"access"`
	AuthorizedPubkey string `json:"authorized_pubkey"`
}

// fetchRepositoryInfo fetches repository information from the server
func fetchRepositoryInfo(url, token string) (*RepositoryInfo, error) {
	// Extract the repository ID from the URL
	repoID := extractRepoID(url)
	
	// Construct the base server URL
	serverBaseURL := extractServerBaseURL(url)
	infoURL := fmt.Sprintf("%s/api/mgit/repos/%s/info", serverBaseURL, repoID)
	
	// Create the request
	req, err := http.NewRequest("GET", infoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add the authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	
	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check the response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from server: %s", string(bodyBytes))
	}
	
	// Parse the response
	var repoInfo RepositoryInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	
	return &repoInfo, nil
}

// extractRepoID extracts the repository ID from a URL
func extractRepoID(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

// extractServerBaseURL extracts the base server URL from a repository URL
func extractServerBaseURL(url string) string {
	// Remove the repo ID and any trailing path
	lastSlashIndex := strings.LastIndex(url, "/")
	if lastSlashIndex == -1 {
		return url
	}
	
	return url[:lastSlashIndex]
}

// cloneGitData clones the Git data using git-upload-pack
func cloneGitData(url, destination, token string) error {
	// Extract the repository ID and server base URL
	repoID := extractRepoID(url)
	serverBaseURL := extractServerBaseURL(url)
	
	// For now, we'll use the git command with a custom header to clone the repository
	// In a real implementation, we would use go-git or a similar library
	
// 	// Create a temporary config file to include the authorization header
// 	tempConfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("mgit-clone-%d.tmp", os.Getpid()))
// 	defer os.Remove(tempConfigPath)
	
// 	configContent := fmt.Sprintf(`[http]
// 	extraHeader = Authorization: Bearer %s
// `, token)
	
// 	if err := os.WriteFile(tempConfigPath, []byte(configContent), 0600); err != nil {
// 		return fmt.Errorf("error creating temporary config file: %w", err)
// 	}
	
	// Construct the Git URL for the upload-pack endpoint
	// gitURL := fmt.Sprintf("%s/api/mgit/repos/%s/git-upload-pack", serverBaseURL, repoID)
	gitURL := fmt.Sprintf("%s/api/mgit/repos/%s", serverBaseURL, repoID)

	// Use git clone with the -c option for Authorization header
	authHeader := fmt.Sprintf("http.extraHeader=Authorization: Bearer %s", token)
	// Debug print statements
	fmt.Println("Debug info for git clone:")
	fmt.Printf("  Auth header config: %s\n", authHeader)
	fmt.Printf("  Token: %s\n", token)
	fmt.Printf("  Git URL: %s\n", gitURL)
	fmt.Printf("  Destination: %s\n", destination)
	
	// Use git clone with the temporary config
	cmd := exec.Command("git", "clone", "-c", authHeader, gitURL, destination)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running git clone: %w", err)
	}
	
	return nil
}

// fetchMGitMetadata fetches the MGit metadata and sets it up in the repository
func fetchMGitMetadata(url, destination, token string) error {
	// Extract the repository ID and server base URL
	repoID := extractRepoID(url)
	serverBaseURL := extractServerBaseURL(url)
	
	// Construct the URL for the MGit metadata endpoint
	metadataURL := fmt.Sprintf("%s/api/mgit/repos/%s/metadata", serverBaseURL, repoID)
	
	// Create the request
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
			return fmt.Errorf("error creating request: %w", err)
	}
	
	// Add the authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	
	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
			return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check the response status
	if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("error response from server: %s", string(bodyBytes))
	}
	
	// Parse the response to get the mappings
	var mappings []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mappings); err != nil {
			return fmt.Errorf("error parsing metadata response: %w", err)
	}
	
	// Create the .mgit directory structure
	mgitDir := filepath.Join(destination, ".mgit")
	mappingsDir := filepath.Join(mgitDir, "mappings")
	if err := os.MkdirAll(mappingsDir, 0755); err != nil {
			return fmt.Errorf("error creating .mgit/mappings directory: %w", err)
	}
	
	// Write the hash_mappings.json file
	mappingsPath := filepath.Join(mappingsDir, "hash_mappings.json")
	mappingsJSON, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
			return fmt.Errorf("error serializing mappings: %w", err)
	}
	
	if err := os.WriteFile(mappingsPath, mappingsJSON, 0644); err != nil {
			return fmt.Errorf("error writing hash_mappings.json file: %w", err)
	}
	
	fmt.Printf("Successfully fetched and stored MGit metadata\n")
	return nil
}

// setupMGitConfig sets up the MGit configuration for the cloned repository
func setupMGitConfig(destination string, repoInfo *RepositoryInfo) error {
	// Create the MGit config
	configPath := filepath.Join(destination, ".mgit", "config")
	
	// Load existing config if it exists, or create a new one
	var config *Config
	var err error
	
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		config = &Config{
			Sections: make(map[string]map[string]string),
		}
	} else {
		config, err = LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("error loading MGit config: %w", err)
		}
	}
	
	// Set the repository information
	config.Set("repository", "id", repoInfo.ID)
	config.Set("repository", "name", repoInfo.Name)
	
	// Save the config
	if err := config.Save(configPath); err != nil {
		return fmt.Errorf("error saving MGit config: %w", err)
	}
	
	return nil
}