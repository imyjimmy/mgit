package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MGitObjectType represents the type of MGit object
type MGitObjectType string

const (
	MGitCommitObject MGitObjectType = "commit"
	MGitTreeObject   MGitObjectType = "tree"
	MGitBlobObject   MGitObjectType = "blob"
)

// Represents an mcommit object
type MCommitStruct struct {
	Type         MGitObjectType       `json:"type"`
	MGitHash     string               `json:"mgit_hash"`
	GitHash      string               `json:"git_hash"`
	TreeHash     string               `json:"tree_hash"`
	ParentHashes []string             `json:"parent_hashes"` // MGit hashes of parents
	Author       *MGitSignature       `json:"author"`
	Committer    *MGitSignature       `json:"committer"`
	Message      string               `json:"message"`
	Metadata     map[string]string    `json:"metadata,omitempty"` // For extensibility
}

// MGitSignature represents a signature in an MGit commit
type MGitSignature struct {
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Pubkey string    `json:"pubkey,omitempty"`
	When   time.Time `json:"when"`
}

// MGitStorage handles the storage and retrieval of MGit objects
type MGitStorage struct {
	RootDir string // Usually ".mgit"
}

// NewMGitStorage creates a new storage instance
func NewMGitStorage() *MGitStorage {
	return &MGitStorage{
		RootDir: ".mgit",
	}
}

// Initialize creates the necessary directory structure for MGit
func (s *MGitStorage) Initialize() error {
	// Create the main directory
	if err := os.MkdirAll(s.RootDir, 0755); err != nil {
		return fmt.Errorf("failed to create MGit directory: %w", err)
	}
	
	// Create subdirectories
	dirs := []string{
		filepath.Join(s.RootDir, "objects"),  // For storing commit objects
		filepath.Join(s.RootDir, "refs"),     // For storing branch refs
		filepath.Join(s.RootDir, "refs/heads"), // For branch heads
		filepath.Join(s.RootDir, "refs/tags"),  // For tags
		filepath.Join(s.RootDir, "mappings"), // For storing hash mappings
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create an initial HEAD file if it doesn't exist
	headPath := filepath.Join(s.RootDir, "HEAD")
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		// Default to "ref: refs/heads/master"
		if err := ioutil.WriteFile(headPath, []byte("ref: refs/heads/master"), 0644); err != nil {
			return fmt.Errorf("failed to create HEAD file: %w", err)
		}
	}
	
	return nil
}

// StoreCommit stores an MGit commit object
func (s *MGitStorage) StoreCommit(commit *MCommitStruct) error {
	// Ensure the hash is set
	if commit.MGitHash == "" {
		return fmt.Errorf("MGit hash cannot be empty")
	}
	
	// Set the object type
	commit.Type = MGitCommitObject
	
	// Create the object path using the hash
	prefix := commit.MGitHash[:2]
	suffix := commit.MGitHash[2:]
	objDir := filepath.Join(s.RootDir, "objects", prefix)
	objPath := filepath.Join(objDir, suffix)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(objDir, 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}
	
	// Write to file
	if err := ioutil.WriteFile(objPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write commit object: %w", err)
	}
	
	return nil
}

// GetCommit retrieves an MGit commit by hash
func (s *MGitStorage) GetCommit(mgitHash string) (*MCommitStruct, error) {
	if len(mgitHash) < 4 {
		return nil, fmt.Errorf("MGit hash too short, need at least 4 characters")
	}
	
	// Handle abbreviated hashes by searching
	if len(mgitHash) < 40 {
		matches, err := s.findObjectByPrefix(mgitHash)
		if err != nil {
			return nil, err
		}
		
		if len(matches) == 0 {
			return nil, fmt.Errorf("no object found with hash prefix %s", mgitHash)
		}
		
		if len(matches) > 1 {
			return nil, fmt.Errorf("ambiguous hash prefix %s matches multiple objects", mgitHash)
		}
		
		mgitHash = matches[0]
	}
	
	// Get the object path
	prefix := mgitHash[:2]
	suffix := mgitHash[2:]
	objPath := filepath.Join(s.RootDir, "objects", prefix, suffix)
	
	// Check if the file exists
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("commit object not found: %s", mgitHash)
	}
	
	// Read the file
	data, err := ioutil.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit object: %w", err)
	}
	
	// Unmarshal from JSON
	var commit MCommitStruct
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit: %w", err)
	}
	
	return &commit, nil
}

// findObjectByPrefix finds objects that start with the given prefix
func (s *MGitStorage) findObjectByPrefix(prefix string) ([]string, error) {
	matches := []string{}
	
	// For very short prefixes (1-2 chars), search directory names
	if len(prefix) <= 2 {
		objDir := filepath.Join(s.RootDir, "objects", prefix)
		if _, err := os.Stat(objDir); os.IsNotExist(err) {
			return matches, nil
		}
		
		files, err := ioutil.ReadDir(objDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read object directory: %w", err)
		}
		
		for _, file := range files {
			matches = append(matches, prefix+file.Name())
		}
		return matches, nil
	}
	
	// For longer prefixes, check the first 2 chars and then match on files
	dirPrefix := prefix[:2]
	filePrefix := prefix[2:]
	objDir := filepath.Join(s.RootDir, "objects", dirPrefix)
	
	if _, err := os.Stat(objDir); os.IsNotExist(err) {
		return matches, nil
	}
	
	files, err := ioutil.ReadDir(objDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read object directory: %w", err)
	}
	
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			matches = append(matches, dirPrefix+file.Name())
		}
	}
	
	return matches, nil
}

// UpdateRef updates an MGit reference (branch or tag)
func (s *MGitStorage) UpdateRef(refName string, mgitHash string) error {
	// Ensure refName is formatted correctly
	if !strings.HasPrefix(refName, "refs/") {
		refName = "refs/heads/" + refName
	}
	
	refPath := filepath.Join(s.RootDir, refName)
	
	// Create directory if it doesn't exist
	refDir := filepath.Dir(refPath)
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}
	
	// Write the ref
	if err := ioutil.WriteFile(refPath, []byte(mgitHash), 0644); err != nil {
		return fmt.Errorf("failed to write ref: %w", err)
	}
	
	return nil
}

// GetRef gets the MGit hash that a reference points to
func (s *MGitStorage) GetRef(refName string) (string, error) {
	// Ensure refName is formatted correctly
	if !strings.HasPrefix(refName, "refs/") {
		refName = "refs/heads/" + refName
	}
	
	refPath := filepath.Join(s.RootDir, refName)
	
	// Check if the file exists
	if _, err := os.Stat(refPath); os.IsNotExist(err) {
		return "", fmt.Errorf("reference not found: %s", refName)
	}
	
	// Read the ref
	data, err := ioutil.ReadFile(refPath)
	if err != nil {
		return "", fmt.Errorf("failed to read ref: %w", err)
	}
	
	return string(data), nil
}

// UpdateHead updates the HEAD reference
func (s *MGitStorage) UpdateHead(refName string) error {
	headPath := filepath.Join(s.RootDir, "HEAD")
	
	// Format the content as "ref: refs/heads/branch-name"
	// Ensure refName is formatted correctly
	if !strings.HasPrefix(refName, "refs/") {
		refName = "refs/heads/" + refName
	}
	
	content := fmt.Sprintf("ref: %s", refName)
	
	// Write the HEAD file
	if err := ioutil.WriteFile(headPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}
	
	return nil
}

// GetHead gets the current HEAD reference
func (s *MGitStorage) GetHead() (string, error) {
	headPath := filepath.Join(s.RootDir, "HEAD")
	
	// Check if the file exists
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		return "", fmt.Errorf("HEAD not found")
	}
	
	// Read the HEAD file
	data, err := ioutil.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}
	
	// Parse the content
	content := string(data)
	if strings.HasPrefix(content, "ref: ") {
		// It's a reference, return the ref name
		return strings.TrimPrefix(content, "ref: "), nil
	} else {
		// It's a direct hash (detached HEAD)
		return content, nil
	}
}

// GetHeadCommit gets the commit that HEAD points to
func (s *MGitStorage) GetHeadCommit() (*MCommitStruct, error) {
	head, err := s.GetHead()
	if err != nil {
		return nil, err
	}
	
	if strings.HasPrefix(head, "refs/") {
		// It's a reference, get the hash it points to
		hash, err := s.GetRef(head)
		if err != nil {
			return nil, err
		}
		
		// Get the commit object
		return s.GetCommit(hash)
	} else {
		// It's a direct hash
		return s.GetCommit(head)
	}
}

// StoreMapping stores a mapping between Git and MGit hashes
func (s *MGitStorage) StoreMapping(gitHash string, mgitHash string, pubkey string) error {
	mappingPath := filepath.Join(s.RootDir, "mappings", "hash_mappings.json")
	
	// Create directory if it doesn't exist
	mappingDir := filepath.Dir(mappingPath)
	if err := os.MkdirAll(mappingDir, 0755); err != nil {
		return fmt.Errorf("failed to create mapping directory: %w", err)
	}
	
	// Read existing mappings if they exist
	var mappings []struct {
		GitHash  string `json:"git_hash"`
		MGitHash string `json:"mgit_hash"`
		Pubkey   string `json:"pubkey"`
	}
	
	if _, err := os.Stat(mappingPath); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(mappingPath)
		if err != nil {
			return fmt.Errorf("failed to read hash mappings: %w", err)
		}
		
		if err := json.Unmarshal(data, &mappings); err != nil {
			return fmt.Errorf("failed to unmarshal hash mappings: %w", err)
		}
	}
	
	// Add or update the mapping
	newMapping := struct {
		GitHash  string `json:"git_hash"`
		MGitHash string `json:"mgit_hash"`
		Pubkey   string `json:"pubkey"`
	}{
		GitHash:  gitHash,
		MGitHash: mgitHash,
		Pubkey:   pubkey,
	}
	
	// Check for existing mapping
	found := false
	for i, mapping := range mappings {
		if mapping.GitHash == gitHash || mapping.MGitHash == mgitHash {
			mappings[i] = newMapping
			found = true
			break
		}
	}
	
	// Add if not found
	if !found {
		mappings = append(mappings, newMapping)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hash mappings: %w", err)
	}
	
	// Write to file
	if err := ioutil.WriteFile(mappingPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write hash mappings: %w", err)
	}
	
	return nil
}

// GetMappings gets all hash mappings
func (s *MGitStorage) GetMappings() ([]struct {
	GitHash  string `json:"git_hash"`
	MGitHash string `json:"mgit_hash"`
	Pubkey   string `json:"pubkey"`
}, error) {
	mappingPath := filepath.Join(s.RootDir, "mappings", "hash_mappings.json")
	
	var mappings []struct {
		GitHash  string `json:"git_hash"`
		MGitHash string `json:"mgit_hash"`
		Pubkey   string `json:"pubkey"`
	}
	
	// Check if the file exists
	if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
		return mappings, nil // Return empty mappings
	}
	
	// Read the mappings
	data, err := ioutil.ReadFile(mappingPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read hash mappings: %w", err)
	}
	
	if err := json.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hash mappings: %w", err)
	}
	
	return mappings, nil
}

// GetMGitHashFromGit gets the MGit hash for a Git hash
func (s *MGitStorage) GetMGitHashFromGit(gitHash string) (string, error) {
	mappings, err := s.GetMappings()
	if err != nil {
		return "", err
	}
	
	for _, mapping := range mappings {
		if mapping.GitHash == gitHash {
			return mapping.MGitHash, nil
		}
	}
	
	return "", fmt.Errorf("no MGit hash found for Git hash %s", gitHash)
}

// GetGitHashFromMGit gets the Git hash for an MGit hash
func (s *MGitStorage) GetGitHashFromMGit(mgitHash string) (string, error) {
	mappings, err := s.GetMappings()
	if err != nil {
		return "", err
	}
	
	for _, mapping := range mappings {
		if mapping.MGitHash == mgitHash {
			return mapping.GitHash, nil
		}
	}
	
	return "", fmt.Errorf("no Git hash found for MGit hash %s", mgitHash)
}

// GetPubkeyForCommit gets the nostr pubkey for a commit (Git or MGit hash)
func (s *MGitStorage) GetPubkeyForCommit(hash string) (string, error) {
	mappings, err := s.GetMappings()
	if err != nil {
		return "", err
	}
	
	for _, mapping := range mappings {
		if mapping.GitHash == hash || mapping.MGitHash == hash {
			return mapping.Pubkey, nil
		}
	}
	
	return "", fmt.Errorf("no pubkey found for hash %s", hash)
}
