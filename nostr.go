package main

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
)

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