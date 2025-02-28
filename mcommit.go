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

// convertToMGitSignature converts go-git's object.Signature to our MGitSignature
func convertToMGitSignature(sig object.Signature, pubkey string) *MGitSignature {
	return &MGitSignature{
			Name:   sig.Name,
			Email:  sig.Email,
			Pubkey: pubkey,
			When:   sig.When,
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
	gitHash, err := w.Commit(message, commitOpts)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error committing: %s", err)
	}
	
	// If no pubkey is present, just return the Git hash
	if opts.Author.Pubkey == "" {
		return gitHash, nil
	}
	
	// Get the commit object we just created
	gitCommit, err := repo.CommitObject(gitHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error retrieving commit: %w", err)
	}
	
	// Initialize MGit storage
	storage := NewMGitStorage()
	if err := storage.Initialize(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error initializing MGit storage: %w", err)
	}
	
	// Collect MGit hashes for parent commits
	parentMGitHashes := []string{}
	for _, parentGitHash := range gitCommit.ParentHashes {
		mgitHash, err := storage.GetMGitHashFromGit(parentGitHash.String())
		if err == nil {
			// We found an MGit hash for this parent
			parentMGitHashes = append(parentMGitHashes, mgitHash)
			fmt.Printf("Found MGit hash for parent %s: %s\n", 
				parentGitHash.String()[:7], mgitHash[:7])
		} else {
			// No MGit hash found, use the Git hash as a fallback
			parentMGitHashes = append(parentMGitHashes, parentGitHash.String())
			fmt.Printf("No MGit hash found for parent %s\n", parentGitHash.String()[:7])
		}
	}
	
	// Compute the MGit hash
	mgitHash := computeMGitHash(gitCommit, parentMGitHashes, opts.Author.Pubkey)
	
	// Create an MGit commit object
	mgitCommit := &MCommitStruct{
		Type:         MGitCommitObject,
		MGitHash:     mgitHash.String(),
		GitHash:      gitHash.String(),
		TreeHash:     gitCommit.TreeHash.String(),
		ParentHashes: parentMGitHashes,
		Author:       convertToMGitSignature(gitCommit.Author, opts.Author.Pubkey),
		Committer:    convertToMGitSignature(gitCommit.Committer, opts.Author.Pubkey), // assume Author == Committer for now
		Message:      gitCommit.Message,
		Metadata:     map[string]string{"version": "1.0"},
	}
	
	// Store the MGit commit object
	if err := storage.StoreCommit(mgitCommit); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error storing MGit commit: %w", err)
	}
	
	// Store the mapping between Git and MGit hashes
	if err := storage.StoreMapping(gitHash.String(), mgitHash.String(), opts.Author.Pubkey); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("error storing hash mapping: %w", err)
	}
	
	// Update the current branch reference in MGit
	head, err := repo.Head()
	if err == nil && head.Name().IsBranch() {
		branchName := head.Name().Short()
		refName := fmt.Sprintf("refs/heads/%s", branchName)
		
		if err := storage.UpdateRef(refName, mgitHash.String()); err != nil {
			fmt.Printf("Warning: Failed to update branch ref: %s\n", err)
		}
	}
	
	fmt.Printf("Created MGit commit: %s (Git hash: %s)\n", 
		mgitHash.String(), gitHash.String())
	
	return mgitHash, nil
}

// computeMGitHash computes a new hash incorporating the nostr pubkey
// and using parent MGit hashes instead of Git hashes
func computeMGitHash(commit *object.Commit, parentMGitHashes []string, pubkey string) plumbing.Hash {
	// Create a new hasher
	hasher := sha1.New()
	
	// Include the tree hash
	hasher.Write(commit.TreeHash[:])
	
	// Include all parent MGit hashes
	for _, parentHashStr := range parentMGitHashes {
		parentHash := plumbing.NewHash(parentHashStr)
		hasher.Write(parentHash[:])
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
		commit.Committer.When.Unix(),
		pubkey)
	hasher.Write([]byte(committerStr))
	
	// Include the commit message
	hasher.Write([]byte(committerStr))
	
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