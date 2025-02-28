package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// HandleShow handles the show command
func HandleShow(args []string) {
	// Default to HEAD if no argument provided
	commitRef := "HEAD"
	if len(args) > 0 {
		commitRef = args[0]
	}

	repo := getRepo()

	// Try to resolve the reference
	hash, err := resolveRevision(repo, commitRef)
	if err != nil {
		fmt.Printf("Error resolving reference '%s': %s\n", commitRef, err)
		os.Exit(1)
	}

	// Get the commit object
	commit, err := repo.CommitObject(hash)
	if err != nil {
		fmt.Printf("Error getting commit: %s\n", err)
		os.Exit(1)
	}

	// Display commit information
	displayCommit(commit)

	// Show the diff for this commit
	showCommitDiff(repo, commit)
}

// HandleMGitShow handles the mgit show command, showing a specific MGit commit
func HandleMGitShow(args []string) {
	if len(args) < 1 {
			fmt.Println("Usage: mgit show <hash>")
			os.Exit(1)
	}

	hash := args[0]
	storage := NewMGitStorage()

	// Get the MGit commit
	mgitCommit, err := storage.GetCommit(hash)
	if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
	}

	// Print the MGit commit details
	printMGitCommit(mgitCommit)

	// Show parent information
	if len(mgitCommit.ParentHashes) > 0 {
			fmt.Println("Parents:")
			for _, parent := range mgitCommit.ParentHashes {
					fmt.Printf("  %s\n", parent)
			}
			fmt.Println()
	}

	// Get the corresponding Git hash
	gitHash := mgitCommit.GitHash
	if gitHash == "" {
			fmt.Println("No Git hash found for this MGit commit")
			return
	}

	// Get repository
	repo := getRepo()

	// Get the Git commit object
	gitCommitHash := plumbing.NewHash(gitHash)
	gitCommit, err := repo.CommitObject(gitCommitHash)
	if err != nil {
			fmt.Printf("Error getting Git commit: %s\n", err)
			return
	}

	// Show the diff using the existing function
	showCommitDiff(repo, gitCommit)
}

// resolveRevision resolves a revision (branch, tag, commit hash) to a commit hash
func resolveRevision(repo *git.Repository, rev string) (plumbing.Hash, error) {
	// If it's HEAD, resolve it
	if rev == "HEAD" {
			ref, err := repo.Head()
			if err == nil {
					return ref.Hash(), nil
			}
	}

	// Try to resolve as a reference (branch, tag)
	ref, err := repo.Reference(plumbing.ReferenceName(rev), true)
	if err == nil {
			return ref.Hash(), nil
	}

	// Try with refs/heads/ prefix
	ref, err = repo.Reference(plumbing.ReferenceName("refs/heads/"+rev), true)
	if err == nil {
			return ref.Hash(), nil
	}

	// Try with refs/tags/ prefix
	ref, err = repo.Reference(plumbing.ReferenceName("refs/tags/"+rev), true)
	if err == nil {
			return ref.Hash(), nil
	}

	// If it's a full 40-char hash
	if len(rev) == 40 && plumbing.IsHash(rev) {
			hash := plumbing.NewHash(rev)
			_, err := repo.CommitObject(hash)
			if err == nil {
					return hash, nil
			}
	}

	// If it's a partial hash, try to find a matching commit
	if len(rev) >= 4 && len(rev) < 40 {
			// List all commits and find a match
			iter, err := repo.CommitObjects()
			if err != nil {
					return plumbing.ZeroHash, fmt.Errorf("error listing commits: %s", err)
			}
			defer iter.Close()

			var matchingHash plumbing.Hash
			matchCount := 0

			err = iter.ForEach(func(c *object.Commit) error {
					if strings.HasPrefix(c.Hash.String(), rev) {
							matchingHash = c.Hash
							matchCount++
					}
					return nil
			})

			if err != nil {
					return plumbing.ZeroHash, fmt.Errorf("error searching commits: %s", err)
			}

			if matchCount == 1 {
					return matchingHash, nil
			} else if matchCount > 1 {
					return plumbing.ZeroHash, fmt.Errorf("ambiguous commit hash prefix: %s", rev)
			}
	}

	// Check nostr mappings for MGit hashes
	if pubkey := GetNostrPubKey(); pubkey != "" {
			// Read all mappings and search for matches
			mappings := getAllNostrMappings()
			if len(mappings) > 0 {
				// fmt.Printf("Checking %d nostr mappings for hash '%s'\n", len(mappings), rev)

				for _, mapping := range mappings {
					// fmt.Printf("Comparing with: Git=%s, MGit=%s\n", mapping.GitHash, mapping.MGitHash)
					
					// Check for exact MGitHash match
					if mapping.MGitHash == rev {
							// fmt.Printf("Found mapping: MGit=%s -> Git=%s\n", rev, mapping.GitHash)
							return plumbing.NewHash(mapping.GitHash), nil
					}
					
					// Check for prefix match if it's a partial hash
					if len(rev) >= 4 && len(rev) < 40 && strings.HasPrefix(mapping.MGitHash, rev) {
							// fmt.Printf("Found mapping for partial hash: MGit=%s -> Git=%s\n", mapping.MGitHash, mapping.GitHash)
							return plumbing.NewHash(mapping.GitHash), nil
					}
				}
			}
			fmt.Printf("No matching MGit hash found in mappings\n")
	} else { fmt.Printf("no nostr pubkey!") }
	return plumbing.ZeroHash, fmt.Errorf("revision not found")
}

// displayCommit shows formatted commit information
func displayCommit(commit *object.Commit) {
	// Get the MGit hash for this commit
	mgitHash := GetMGitHashForCommit(commit.Hash)
	
	// If we have an MGit hash, display that
	if mgitHash != "" {
			fmt.Printf("commit %s\n", mgitHash)
	} else {
			// Otherwise fall back to the Git hash
			fmt.Printf("commit %s\n", commit.Hash.String())
	}
	
	// Get the nostr pubkey for this commit
	pubkey := GetCommitNostrPubkey(commit.Hash)
	
	// Display author with pubkey in the format requested
	if pubkey != "" {
			fmt.Printf("Author: %s <%s> <%s>\n", commit.Author.Name, commit.Author.Email, pubkey)
	} else {
			fmt.Printf("Author: %s <%s>\n", commit.Author.Name, commit.Author.Email)
	}
	
	fmt.Printf("Date:   %s\n\n", commit.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700"))

	// Print the commit message with indentation
	for _, line := range strings.Split(commit.Message, "\n") {
			fmt.Printf("    %s\n", line)
	}
	fmt.Println()
}

// showCommitDiff shows the diff for a commit using git's diff command
func showCommitDiff(repo *git.Repository, commit *object.Commit) {
	// Get the repository path
	wt, err := repo.Worktree()
	if err != nil {
			fmt.Printf("Error getting worktree: %s\n", err)
			return
	}
	repoPath := wt.Filesystem.Root()

	// Prepare git command to show the diff
	var cmd *exec.Cmd
	var args []string

	// For commits with a parent, we don't need to handle the parent specially
	// git show will automatically compare with the parent
	args = []string{"-C", repoPath, "show", "--no-color", "--patch", commit.Hash.String()}
	
	cmd = exec.Command("git", args...)
	
	// Run the command and capture output
	output, err := cmd.Output()
	if err != nil {
			fmt.Printf("Error executing git diff: %s\n", err)
			if exitErr, ok := err.(*exec.ExitError); ok {
					fmt.Printf("git diff stderr: %s\n", string(exitErr.Stderr))
			}
			return
	}
	
	// Extract just the diff part (after the commit information)
	diffOutput := string(output)
	diffStart := strings.Index(diffOutput, "diff --git")
	if diffStart >= 0 {
			diffOutput = diffOutput[diffStart:]
	}
	
	// Print the diff
	fmt.Println(diffOutput)
}

// displayFileDiff shows the diff for a single file change
func displayFileDiff(change *object.Change) {
	from, to, err := change.Files()
	if err != nil {
		fmt.Printf("Error getting file info: %s\n", err)
		return
	}
	
	if from == nil && to == nil {
		return
	}

	// Get file names
	var fromName, toName string
	if from != nil {
		fromName = from.Name
	}
	if to != nil {
		toName = to.Name
	}

	// Handle renamed files
	if fromName != toName && from != nil && to != nil {
		fmt.Printf("diff --git a/%s b/%s\n", fromName, toName)
		fmt.Printf("rename from %s\n", fromName)
		fmt.Printf("rename to %s\n", toName)
	} else {
		// Regular file change
		fmt.Printf("diff --git a/%s b/%s\n", fromName, toName)
	}

	// Handle file mode changes
	if from != nil && to != nil && from.Mode != to.Mode {
		fmt.Printf("old mode %o\n", from.Mode)
		fmt.Printf("new mode %o\n", to.Mode)
	}

	// Handle new or deleted files
	if from == nil {
		fmt.Printf("new file mode %o\n", to.Mode)
		fmt.Printf("--- /dev/null\n")
		fmt.Printf("+++ b/%s\n", toName)

		content, err := to.Contents()
		if err != nil {
			fmt.Printf("Error getting file contents: %s\n", err)
			return
		}

		fmt.Println("@@ -0,0 +1," + fmt.Sprintf("%d", len(strings.Split(content, "\n"))) + " @@")
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				fmt.Printf("+%s\n", line)
			}
		}
	} else if to == nil {
		fmt.Printf("deleted file mode %o\n", from.Mode)
		fmt.Printf("--- a/%s\n", fromName)
		fmt.Printf("+++ /dev/null\n")

		content, err := from.Contents()
		if err != nil {
			fmt.Printf("Error getting file contents: %s\n", err)
			return
		}

		fmt.Println("@@ -1," + fmt.Sprintf("%d", len(strings.Split(content, "\n"))) + " +0,0 @@")
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				fmt.Printf("-%s\n", line)
			}
		}
	} else {
		// Modified file - compute the diff
    fmt.Printf("--- a/%s\n", fromName)
    fmt.Printf("+++ b/%s\n", toName)

    // Get file contents
    fromContent, err := from.Contents()
    if err != nil {
        fmt.Printf("Error getting file contents: %s\n", err)
        return
    }

    toContent, err := to.Contents()
    if err != nil {
        fmt.Printf("Error getting file contents: %s\n", err)
        return
    }

    // Show complete diff of the files
    fromLines := strings.Split(fromContent, "\n")
    toLines := strings.Split(toContent, "\n")

    fmt.Printf("@@ -1,%d +1,%d @@\n", len(fromLines), len(toLines))
    
    // Show all lines from the old file with - prefix
    for _, line := range fromLines {
        if line != "" {
            fmt.Printf("-%s\n", line)
        }
    }
    
    // Show all lines from the new file with + prefix
    for _, line := range toLines {
        if line != "" {
            fmt.Printf("+%s\n", line)
        }
    }
	}
	fmt.Println()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}