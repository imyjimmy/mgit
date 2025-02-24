package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// NostrCommitMapping represents the mapping between commit hashes and nostr pubkeys
type NostrCommitMapping struct {
	GitHash  string `json:"git_hash"`
	MGitHash string `json:"mgit_hash"`
	Pubkey   string `json:"pubkey"`
}

// GetNostrPubKey gets the user's nostr public key
func GetNostrPubKey() string {
	return GetConfigValue("user.pubkey", "")
}

// HasNostrPubKey checks if the user has a nostr public key configured
func HasNostrPubKey() bool {
	return GetNostrPubKey() != ""
}

// ValidateNostrPubKey validates a nostr public key
func ValidateNostrPubKey(pubkey string) bool {
	// Basic validation - ensure it starts with "npub" and is of the right length
	// You could add more sophisticated validation here if needed
	return strings.HasPrefix(pubkey, "npub") && len(pubkey) >= 60
}

// SignWithNostrKey is a placeholder for future implementation
// This function could be used later when you want to sign commits with the nostr key
func SignWithNostrKey(message string) (string, error) {
	pubkey := GetNostrPubKey()
	if pubkey == "" {
		return "", fmt.Errorf("no nostr public key configured")
	}
	
	// In a real implementation, you'd use the private key to sign the message
	// For now, we'll just return a placeholder
	return fmt.Sprintf("nostr-signed:%s:%s", pubkey, message), nil
}

// VerifyNostrSignature is a placeholder for future implementation
func VerifyNostrSignature(message, signature, pubkey string) bool {
	// In a real implementation, you'd verify the signature
	// For now, we'll just return a placeholder
	expectedSig := fmt.Sprintf("nostr-signed:%s:%s", pubkey, message)
	return signature == expectedSig
}

// AddNostrMetadataToCommit is a conceptual example for future implementation
func AddNostrMetadataToCommit(commit *object.Commit) *object.Commit {
	// This is just a conceptual example - the go-git library might not allow
	// direct modification of commit objects like this
	pubkey := GetNostrPubKey()
	if pubkey != "" {
		// In a real implementation, you would add the pubkey as
		// extra metadata to the commit
	}
	return commit
}

// GetCommitNostrPubkey retrieves the nostr pubkey associated with a commit
func GetCommitNostrPubkey(hash plumbing.Hash) string {
	// Get the mapping file path
	mappingFile := getNostrMappingFilePath()
	
	// Check if the mapping file exists
	if _, err := os.Stat(mappingFile); os.IsNotExist(err) {
		return "" // No mapping file exists yet
	}
	
	// Read the mapping file
	data, err := os.ReadFile(mappingFile)
	if err != nil {
		fmt.Printf("Warning: Error reading nostr mapping file: %s\n", err)
		return ""
	}
	
	// Parse the mappings
	var mappings []NostrCommitMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		fmt.Printf("Warning: Error parsing nostr mapping file: %s\n", err)
		return ""
	}
	
	// Look for the commit hash in the mappings
	hashStr := hash.String()
	for _, mapping := range mappings {
		if mapping.GitHash == hashStr || mapping.MGitHash == hashStr {
			return mapping.Pubkey
		}
	}
	
	// If we didn't find a mapping, return empty string
	return ""
}

// StoreCommitNostrMapping stores the mapping between a git commit hash, an mgit hash, and a nostr pubkey
func StoreCommitNostrMapping(gitHash, mgitHash plumbing.Hash, pubkey string) error {
	// Get the mapping file path
	mappingFile := getNostrMappingFilePath()
	
	// Check if the mapping file exists
	var mappings []NostrCommitMapping
	if _, err := os.Stat(mappingFile); !os.IsNotExist(err) {
		// Read existing mappings
		data, err := os.ReadFile(mappingFile)
		if err != nil {
			return fmt.Errorf("error reading nostr mapping file: %s", err)
		}
		
		// Parse existing mappings
		if err := json.Unmarshal(data, &mappings); err != nil {
			return fmt.Errorf("error parsing nostr mapping file: %s", err)
		}
	}
	
	// Add the new mapping
	newMapping := NostrCommitMapping{
		GitHash:  gitHash.String(),
		MGitHash: mgitHash.String(),
		Pubkey:   pubkey,
	}
	
	// Check for duplicates and update if exists
	found := false
	for i, mapping := range mappings {
		if mapping.GitHash == newMapping.GitHash || mapping.MGitHash == newMapping.MGitHash {
			mappings[i] = newMapping
			found = true
			break
		}
	}
	
	// If not found, append
	if !found {
		mappings = append(mappings, newMapping)
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding mapping data: %s", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(mappingFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory for mapping file: %s", err)
	}
	
	// Write to file
	if err := os.WriteFile(mappingFile, data, 0644); err != nil {
		return fmt.Errorf("error writing mapping file: %s", err)
	}
	
	return nil
}

// getNostrMappingFilePath returns the path to the nostr mapping file
func getNostrMappingFilePath() string {
	// Store the mapping in the .mgit directory
	return ".mgit/nostr_mappings.json"
}

// getAllNostrMappings retrieves all nostr commit mappings
func getAllNostrMappings() []NostrCommitMapping {
	// Get the mapping file path
	mappingFile := getNostrMappingFilePath()
	
	// Check if the mapping file exists
	if _, err := os.Stat(mappingFile); os.IsNotExist(err) {
			return []NostrCommitMapping{} // No mapping file exists yet
	}
	
	// Read the mapping file
	data, err := os.ReadFile(mappingFile)
	if err != nil {
			fmt.Printf("Warning: Error reading nostr mapping file: %s\n", err)
			return []NostrCommitMapping{}
	}
	
	// Parse the mappings
	var mappings []NostrCommitMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
			fmt.Printf("Warning: Error parsing nostr mapping file: %s\n", err)
			return []NostrCommitMapping{}
	}
	
	return mappings
}