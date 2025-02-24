package main

import (
	"fmt"
	"os"
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

// resolveRevision resolves a revision (branch, tag, commit hash) to a commit hash
func resolveRevision(repo *git.Repository, rev string) (plumbing.Hash, error) {
	// First, try to resolve as a reference (branch, tag)
	ref, err := repo.Reference(plumbing.ReferenceName(rev), true)
	if err == nil {
		return ref.Hash(), nil
	}

	// If that fails, try as a short or full hash
	if plumbing.IsHash(rev) {
		return plumbing.NewHash(rev), nil
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

	// If it's HEAD, resolve it
	if rev == "HEAD" {
		ref, err := repo.Head()
		if err == nil {
			return ref.Hash(), nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("revision not found")
}

// displayCommit shows formatted commit information
func displayCommit(commit *object.Commit) {
	fmt.Printf("commit %s\n", commit.Hash.String())
	fmt.Printf("Author: %s <%s>\n", commit.Author.Name, commit.Author.Email)
	fmt.Printf("Date:   %s\n\n", commit.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700"))

	// Check if we can get nostr pubkey for this commit
	pubkey := GetCommitNostrPubkey(commit.Hash)
	if pubkey != "" {
		fmt.Printf("Nostr:  %s\n\n", pubkey)
	}

	// Print the commit message with indentation
	for _, line := range strings.Split(commit.Message, "\n") {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()
}

// showCommitDiff shows the diff for a commit
func showCommitDiff(repo *git.Repository, commit *object.Commit) {
	// Get the tree for this commit
	tree, err := commit.Tree()
	if err != nil {
		fmt.Printf("Error getting tree: %s\n", err)
		return
	}

	// Get parent commit (if any)
	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parents().Next()
		if err == nil {
			parentTree, err = parent.Tree()
			if err != nil {
				fmt.Printf("Error getting parent tree: %s\n", err)
				return
			}
		}
	}

	// If we have a parent tree, show the diff
	if parentTree != nil {
		changes, err := object.DiffTree(parentTree, tree)
		if err != nil {
			fmt.Printf("Error computing diff: %s\n", err)
			return
		}

		for _, change := range changes {
			displayFileDiff(change)
		}
	} else {
		// No parent, show the initial commit files
		files := tree.Files()
		
		err = files.ForEach(func(f *object.File) error {
			fmt.Printf("diff --git a/%s b/%s\n", f.Name, f.Name)
			fmt.Printf("new file mode %o\n", f.Mode)
			fmt.Printf("--- /dev/null\n")
			fmt.Printf("+++ b/%s\n", f.Name)

			content, err := f.Contents()
			if err != nil {
				return err
			}

			fmt.Println("@@ -0,0 +1," + fmt.Sprintf("%d", len(strings.Split(content, "\n"))) + " @@")
			for _, line := range strings.Split(content, "\n") {
				if line != "" {
					fmt.Printf("+%s\n", line)
				}
			}
			fmt.Println()
			return nil
		})
		if err != nil {
			fmt.Printf("Error iterating files: %s\n", err)
		}
	}
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

		// Simple line-by-line diff for modified files
		// In a real implementation, you'd want to use a proper diff algorithm
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

		// Very simple diff - just show old and new content
		// In a real implementation, you'd use a proper diff algorithm
		fromLines := strings.Split(fromContent, "\n")
		toLines := strings.Split(toContent, "\n")

		fmt.Printf("@@ -1,%d +1,%d @@\n", len(fromLines), len(toLines))
		
		// For simplicity, just show a few lines with + and -
		// A real implementation would compute actual line differences
		for i := 0; i < min(len(fromLines), 3); i++ {
			if fromLines[i] != "" {
				fmt.Printf("-%s\n", fromLines[i])
			}
		}
		for i := 0; i < min(len(toLines), 3); i++ {
			if toLines[i] != "" {
				fmt.Printf("+%s\n", toLines[i])
			}
		}
		if len(fromLines) > 3 || len(toLines) > 3 {
			fmt.Println("... (diff truncated)")
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