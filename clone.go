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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
		fmt.Println("Usage: mgit clone [-jwt <token>] <url> [destination]")
		os.Exit(1)
	}

	// Parse arguments for -jwt flag
	var jwtToken string
	var url string
	var destination string
	
	// Parse command line arguments
	i := 0
	for i < len(args) {
		if args[i] == "-jwt" {
			if i+1 >= len(args) {
				fmt.Println("Error: -jwt flag requires a token argument")
				fmt.Println("Usage: mgit clone [-jwt <token>] <url> [destination]")
				os.Exit(1)
			}
			jwtToken = args[i+1]
			i += 2 // Skip both -jwt and token
		} else if url == "" {
			url = args[i]
			i++
		} else if destination == "" {
			destination = args[i]
			i++
		} else {
			fmt.Printf("Error: unexpected argument '%s'\n", args[i])
			fmt.Println("Usage: mgit clone [-jwt <token>] <url> [destination]")
			os.Exit(1)
		}
	}

	// Validate that we have at least a URL
	if url == "" {
		fmt.Println("Error: repository URL is required")
		fmt.Println("Usage: mgit clone [-jwt <token>] <url> [destination]")
		os.Exit(1)
	}

	// If no destination is specified, use the last part of the URL as the directory name
	if destination == "" {
		parts := strings.Split(url, "/")
		destination = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	// Normalize URL to ensure it doesn't end with a slash
	url = strings.TrimSuffix(url, "/")

	// Get token for the repository
	var token string
	if jwtToken != "" {
		// Use the provided JWT token
		fmt.Println("Using provided JWT token for authentication")
		token = jwtToken
	} else {
		// Fall back to stored token lookup
		token = getTokenForRepo(url)
	}

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
	fmt.Println("Cloning Git repository...")
	if err := gitClone(url, destination, token); err != nil {
		return fmt.Errorf("error cloning Git repository: %w", err)
	}

	// Fetch and set up MGit metadata
	fmt.Println("Setting up MGit metadata...")
	if err := fetchMGitMetadata(url, destination, token); err != nil {
		// Don't fail the clone if metadata fetch fails - log warning and continue
		fmt.Printf("Warning: Failed to fetch MGit metadata: %s\n", err)
	}

	// Reconstruct MGit objects from mappings
	fmt.Println("Reconstructing MGit objects...")
	if err := reconstructMGitObjects(destination); err != nil {
		// Don't fail the clone if reconstruction fails - log warning and continue
		fmt.Printf("Warning: Failed to reconstruct MGit objects: %s\n", err)
	}

	// Set up MGit configuration
	if err := setupMGitConfig(destination, repoInfo); err != nil {
		return fmt.Errorf("error setting up MGit config: %w", err)
	}

	return nil
}

// RepositoryInfo represents information about a repository
type RepositoryInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Access string `json:"access"`
}

// fetchRepositoryInfo fetches information about the repository
func fetchRepositoryInfo(url, token string) (*RepositoryInfo, error) {
	// Extract the repository ID and server base URL
	repoID := extractRepoID(url)
	serverBaseURL := extractServerBaseURL(url)
	
	// Construct the URL for the repository info endpoint
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
	// Remove trailing slashes and .git suffix
	url = strings.TrimSuffix(strings.TrimSuffix(url, "/"), ".git")
	
	// Get the last part of the URL
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

// extractServerBaseURL extracts the server base URL from a repository URL
func extractServerBaseURL(url string) string {
	// Find the last occurrence of the repository ID
	repoID := extractRepoID(url)
	
	// Remove the repository ID from the end to get the base URL
	baseURL := strings.TrimSuffix(url, "/"+repoID)
	baseURL = strings.TrimSuffix(baseURL, repoID)
	
	return baseURL
}

// gitClone performs the actual Git clone operation
func gitClone(url, destination, token string) error {
	// Extract repository ID and server base URL for the Git endpoint
	repoID := extractRepoID(url)
	serverBaseURL := extractServerBaseURL(url)
	
	// Construct the Git URL - this should point to the Git protocol endpoint
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
	
	// ADDED: Also write to nostr_mappings.json for compatibility
	nostrMappingsPath := filepath.Join(mgitDir, "nostr_mappings.json")
	if err := os.WriteFile(nostrMappingsPath, mappingsJSON, 0644); err != nil {
		return fmt.Errorf("error writing nostr_mappings.json file: %w", err)
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

// reconstructMGitObjects reconstructs MGit objects from Git commits using mappings
func reconstructMGitObjects(repoPath string) error {
	// Create necessary directory structure first
	mgitDir := filepath.Join(repoPath, ".mgit")
	objDir := filepath.Join(mgitDir, "objects")
	refsDir := filepath.Join(mgitDir, "refs")
	refsHeadsDir := filepath.Join(refsDir, "heads")
	mappingsDir := filepath.Join(mgitDir, "mappings")
	
	dirs := []string{objDir, refsDir, refsHeadsDir, mappingsDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating MGit directory structure: %w", err)
		}
	}

	// Open the Git repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("error opening Git repository: %w", err)
	}
	
	// Use the correct path for hash_mappings.json
	hashMappingsPath := filepath.Join(repoPath, ".mgit", "mappings", "hash_mappings.json")
	
	// Check if the mappings file exists
	if _, err = os.Stat(hashMappingsPath); os.IsNotExist(err) {
		return fmt.Errorf("no MGit mappings found in the repository")
	}
	
	// Read the mappings file
	mappingsData, err := os.ReadFile(hashMappingsPath)
	if err != nil {
		return fmt.Errorf("error reading mappings file: %w", err)
	}
	
	// Parse the mappings
	var mappings []NostrCommitMapping
	if err := json.Unmarshal(mappingsData, &mappings); err != nil {
		return fmt.Errorf("error parsing mappings file: %w", err)
	}
	
	// Create the MGit storage
	storage := &MGitStorage{
		RootDir: filepath.Join(repoPath, ".mgit"),
	}
	
	// Initialize the MGit storage
	if err := storage.Initialize(); err != nil {
		return fmt.Errorf("error initializing MGit storage: %w", err)
	}
	
	// Process each mapping to reconstruct MGit objects
	for _, mapping := range mappings {
		// Get the Git commit
		gitHash := plumbing.NewHash(mapping.GitHash)
		commit, err := repo.CommitObject(gitHash)
		if err != nil {
			fmt.Printf("Warning: Could not find Git commit %s: %s\n", mapping.GitHash, err)
			continue
		}
		
		// Create the MGit commit object
		mgitCommit := &MCommitStruct{
			Type:         MGitCommitObject,
			MGitHash:     mapping.MGitHash,
			GitHash:      mapping.GitHash,
			Message:      commit.Message,
			Author: &MGitSignature{
				Name:   commit.Author.Name,
				Email:  commit.Author.Email,
				Pubkey: mapping.Pubkey,
				When:   commit.Author.When,
			},
			Committer: &MGitSignature{
				Name:   commit.Author.Name,
				Email:  commit.Author.Email,
				Pubkey: mapping.Pubkey,
				When:   commit.Author.When,
			},
			ParentHashes: []string{}, // Will be filled in below
			TreeHash:     commit.TreeHash.String(),
		}
		
		// Find parent MGit hashes
		for _, parentGitHash := range commit.ParentHashes {
			for _, parentMapping := range mappings {
				if parentMapping.GitHash == parentGitHash.String() {
					mgitCommit.ParentHashes = append(mgitCommit.ParentHashes, parentMapping.MGitHash)
					break
				}
			}
		}
		
		// Store the MGit commit
		if err := storage.StoreCommit(mgitCommit); err != nil {
			fmt.Printf("Warning: Could not store MGit commit %s: %s\n", mapping.MGitHash, err)
			continue
		}
		
		fmt.Printf("Reconstructed MGit commit: %s (from Git %s)\n", mapping.MGitHash[:7], mapping.GitHash[:7])
	}
	
	// Update branch references to point to MGit hashes
	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("error getting references: %w", err)
	}
	
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			gitHash := ref.Hash().String()
			
			// Find the corresponding MGit hash
			var mgitHash string
			for _, mapping := range mappings {
				if mapping.GitHash == gitHash {
					mgitHash = mapping.MGitHash
					refPath := filepath.Join(repoPath, ".mgit", "refs", "heads", branchName)
					
					if err := storage.UpdateRef(refPath, mapping.MGitHash); err != nil {
						fmt.Printf("Warning: Could not update branch ref %s: %s\n", branchName, err)
					} else {
						fmt.Printf("Set branch reference %s to MGit hash %s\n", branchName, mgitHash[:7])
					}
					break
				}
			}
			
			if mgitHash == "" {
				fmt.Printf("Warning: Could not find MGit hash for branch %s at git hash %s\n", branchName, gitHash)
			}
		}
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("error processing references: %w", err)
	}
	
	// Update HEAD
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("error getting Git HEAD: %w", err)
	}
	
	// Create HEAD file pointing to the current branch
	if head.Name().IsBranch() {
		branchName := head.Name().Short()
		headContent := fmt.Sprintf("ref: refs/heads/%s", branchName)
		headPath := filepath.Join(repoPath, ".mgit", "HEAD")
		
		if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
			return fmt.Errorf("error writing HEAD file: %w", err)
		}
		
		fmt.Printf("Set HEAD to branch: %s\n", branchName)
	} else {
		// Detached HEAD - try to find the corresponding MGit hash
		gitHash := head.Hash().String()
		
		// Find the MGit hash for this Git hash
		var mgitHash string
		for _, mapping := range mappings {
			if mapping.GitHash == gitHash {
				mgitHash = mapping.MGitHash
				break
			}
		}
		
		if mgitHash == "" {
			return fmt.Errorf("could not find MGit hash for detached HEAD at %s", gitHash)
		}
		
		// Write the direct hash as HEAD
		headPath := filepath.Join(repoPath, ".mgit", "HEAD")
		if err := os.WriteFile(headPath, []byte(mgitHash), 0644); err != nil {
			return fmt.Errorf("error writing HEAD file: %w", err)
		}
		
		fmt.Printf("Set HEAD to detached commit: %s\n", mgitHash[:7])
	}
	
	return nil
}