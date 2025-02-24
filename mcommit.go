package main

import (
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Signature represents the author or committer information including nostr pubkey
type Signature struct {
	// Name represents a person name. It is an arbitrary string.
	Name string
	// Email is an email, but it cannot be assumed to be well-formed.
	Email string
	// Pubkey is the nostr public key
	Pubkey string
	// When is the timestamp of the signature.
	When time.Time
}

// MCommitOptions holds information for committing changes with enhanced mgit features
type MCommitOptions struct {
	Author    *Signature
	Committer *Signature
	// Additional fields can be added here if needed
}

// convertToGitSignature converts our Signature to go-git's object.Signature
func convertToGitSignature(sig *Signature) *object.Signature {
	return &object.Signature{
		Name:  sig.Name,
		Email: sig.Email,
		When:  sig.When,
	}
}

// MGitCommit creates a commit that incorporates the nostr pubkey in hash calculation
func MGitCommit(message string, opts *MCommitOptions) (plumbing.Hash, error) {
	// Get repository
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error getting worktree: %s", err)
	}

	// Convert our signature to go-git signature
	author := convertToGitSignature(opts.Author)
	
	// Create a standard commit using go-git
	commitOpts := &git.CommitOptions{
		Author: author,
	}
	
	// If committer is specified, use it
	if opts.Committer != nil {
		commitOpts.Committer = convertToGitSignature(opts.Committer)
	}
	
	// Perform the standard git commit
	hash, err := w.Commit(message, commitOpts)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error committing: %s", err)
	}
	
	// If no pubkey is present, return the standard hash
	if opts.Author.Pubkey == "" {
		return hash, nil
	}
	
	// Get the commit object to access its components
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error retrieving commit: %s", err)
	}
	
	// Now compute a custom hash that incorporates the nostr pubkey
	mHash := computeMGitHash(commit, opts.Author.Pubkey)
	
	// Store the mapping between the git hash, mgit hash, and nostr pubkey
	err = StoreCommitNostrMapping(hash, mHash, opts.Author.Pubkey)
	if err != nil {
		fmt.Printf("Warning: Failed to store nostr mapping: %s\n", err)
		// Continue even if mapping storage fails
	}
	
	fmt.Printf("Created commit mapping: Git hash %s → MGit hash %s (Nostr pubkey: %s)\n", 
    hash.String(), mHash.String(), opts.Author.Pubkey)

	return mHash, nil
}

// computeMGitHash computes a new hash incorporating the nostr pubkey
func computeMGitHash(commit *object.Commit, pubkey string) plumbing.Hash {
	// Create a new hasher
	hasher := sha1.New()
	
	// Include the tree hash
	hasher.Write(commit.TreeHash[:])
	
	// Include all parent hashes
	for _, parent := range commit.ParentHashes {
		hasher.Write(parent[:])
	}
	
	// Include the author information with pubkey
	authorStr := fmt.Sprintf("%s <%s> %d %s", 
		commit.Author.Name, 
		commit.Author.Email, 
		commit.Author.When.Unix(), 
		pubkey)
	hasher.Write([]byte(authorStr))
	
	// Include committer information
	committerStr := fmt.Sprintf("%s <%s> %d", 
		commit.Committer.Name, 
		commit.Committer.Email, 
		commit.Committer.When.Unix())
	hasher.Write([]byte(committerStr))
	
	// Include the commit message
	hasher.Write([]byte(commit.Message))
	
	// Calculate the new hash
	mgitHash := hasher.Sum(nil)
	
	// Convert to plumbing.Hash
	var result plumbing.Hash
	copy(result[:], mgitHash[:20]) // SHA-1 is 20 bytes
	
	return result
}

// StoreMGitCommitMapping stores a mapping between original git hash and mgit hash
// This is a placeholder - in a real implementation, you would need persistent storage
func StoreMGitCommitMapping(gitHash, mgitHash plumbing.Hash) error {
	// Implementation would store the mapping in a database or file
	return nil
}

// getMGitHashForCommit retrieves the MGit hash for a Git commit hash
func GetMGitHashForCommit(gitHash plumbing.Hash) string {
	mappings := getAllNostrMappings()
	gitHashStr := gitHash.String()
	
	for _, mapping := range mappings {
			if mapping.GitHash == gitHashStr {
					return mapping.MGitHash
			}
	}
	
	return ""
}