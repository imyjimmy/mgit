package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

// HandleMGitCommit handles the mgit commit command
func HandleMGitCommit(args []string) {
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

	// Get user information from config
	userName := GetConfigValue("user.name", "")
	userEmail := GetConfigValue("user.email", "")
	userPubkey := GetConfigValue("user.pubkey", "")

	if userName == "" || userEmail == "" {
		fmt.Println("Please set your user name and email first:")
		fmt.Println("  mgit config --global user.name \"Your Name\"")
		fmt.Println("  mgit config --global user.email \"your.email@example.com\"")
		os.Exit(1)
	}

	// Create the commit with MCommit
	hash, err := MGitCommit(message, &MCommitOptions{
		Author: &Signature{
			Name:   userName,
			Email:  userEmail,
			Pubkey: userPubkey,
			When:   time.Now(),
		},
	})

	if err != nil {
		fmt.Printf("Error committing changes: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Committed changes [%s]: %s\n", hash.String()[:7], message)
}

// HandleMGitLog handles the mgit log command for the MGit hash chain
func HandleMGitLog(args []string) {
	// Parse command line flags
	oneline := false
	graph := false
	decorate := false
	all := false
	maxCount := 10 // Default
	
	for _, arg := range args {
			switch arg {
			case "--oneline":
					oneline = true
			case "--graph":
					graph = true
			case "--decorate=short", "--decorate":
					decorate = true
			case "--all":
					all = true
			}
			
			// Handle -n flag for limiting commits
			if strings.HasPrefix(arg, "-n") {
					if len(arg) > 2 {
							fmt.Sscanf(arg[2:], "%d", &maxCount)
					} else {
							// Look for the next argument
							for i, a := range args {
									if a == "-n" && i+1 < len(args) {
											fmt.Sscanf(args[i+1], "%d", &maxCount)
											break
									}
							}
					}
			}
	}

	// Initialize storage
	storage := NewMGitStorage()
	repo := getRepo()

	// Collect starting commits based on flags
	startingCommits := []*MCommitStruct{}

	// Get the HEAD commit
	headCommit, err := storage.GetHeadCommit()
	if err != nil {
			fmt.Printf("Error getting HEAD commit: %s\n", err)
			os.Exit(1)
	}

	// If --all flag is specified, include commits from all branches
	if all {
		// Get all branches
		refs, err := repo.References()
		if err == nil {
				_ = refs.ForEach(func(ref *plumbing.Reference) error {
						if ref.Name().IsBranch() {
								// Skip if this is the current branch (already added as HEAD)
								if headCommit != nil && ref.Hash().String() == headCommit.GitHash {
										return nil
								}
								
								// Get MGit hash for this Git hash
								mgitHash, err := storage.GetMGitHashFromGit(ref.Hash().String())
								if err == nil {
										commit, err := storage.GetCommit(mgitHash)
										if err == nil {
												startingCommits = append(startingCommits, commit)
										}
								}
						}
						return nil
				})
		}
	}
	
	headRef, err := repo.Head()
	currentBranch := ""
	if err == nil && headRef.Name().IsBranch() {
			currentBranch = headRef.Name().Short()
	}

	// If not using special formatting, use the default format
	if !oneline && !graph {
			fmt.Println("MGit Commit History:")
			fmt.Println("====================")
	}

	// Start with head commit
	if oneline {
			printMGitCommitOneline(headCommit, graph, decorate, currentBranch)
	} else {
			printMGitCommit(headCommit)
	}
	count := 1

	// Process parents recursively with a breadth-first approach
	visited := map[string]bool{headCommit.MGitHash: true}
	queue := headCommit.ParentHashes

	for len(queue) > 0 && count < maxCount {
			currentHash := queue[0]
			queue = queue[1:]

			if visited[currentHash] {
					continue
			}

			commit, err := storage.GetCommit(currentHash)
			if err != nil {
					fmt.Printf("Warning: Could not load commit %s: %s\n", currentHash, err)
					continue
			}

			if oneline {
					printMGitCommitOneline(commit, graph, decorate, "")
			} else {
					printMGitCommit(commit)
			}
			count++
			visited[currentHash] = true

			// Add parents to queue
			for _, parent := range commit.ParentHashes {
					if !visited[parent] {
							queue = append(queue, parent)
					}
			}
	}
}

// printMGitCommitOneline prints a single MGit commit in oneline format
func printMGitCommitOneline(commit *MCommitStruct, showGraph bool, decorate bool, branchName string) {
	// First 7 characters of hash (like git)
	shortHash := commit.MGitHash
	if len(shortHash) > 7 {
			shortHash = shortHash[:7]
	}
	
	// Add graph symbol if requested
	prefix := ""
	if showGraph {
			prefix = "* "
	}
	
	// Add decoration if requested
	decoration := ""
	if decorate && branchName != "" {
			decoration = fmt.Sprintf(" (HEAD -> %s)", branchName)
	}
	
	// Get first line of commit message
	message := commit.Message
	if idx := strings.Index(message, "\n"); idx != -1 {
			message = message[:idx]
	}
	
	fmt.Printf("%s%s%s %s\n", prefix, shortHash, decoration, message)
}

// printMGitCommit prints a single MGit commit
func printMGitCommit(commit *MCommitStruct) {
	fmt.Printf("commit %s\n", commit.MGitHash)
	fmt.Printf("git-commit %s\n", commit.GitHash)
	
	pubkeyInfo := ""
	if commit.Author.Pubkey != "" {
			pubkeyInfo = fmt.Sprintf(" <%s>", commit.Author.Pubkey)
	}
	
	fmt.Printf("Author: %s <%s>%s\n", 
			commit.Author.Name, 
			commit.Author.Email,
			pubkeyInfo)
	
	fmt.Printf("Date:   %s\n\n", 
			commit.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700"))
	
	// Print the commit message with indentation
	for _, line := range strings.Split(commit.Message, "\n") {
			fmt.Printf("    %s\n", line)
	}
	
	fmt.Println()
}

// HandleMGitVerify verifies the integrity of the MGit commit chain
func HandleMGitVerify(args []string) {
	storage := NewMGitStorage()
	
	// Get all commits
	headCommit, err := storage.GetHeadCommit()
	if err != nil {
		fmt.Printf("Error getting HEAD commit: %s\n", err)
		os.Exit(1)
	}
	
	// Build the commit graph
	commits := make(map[string]*MCommitStruct)
	visited := make(map[string]bool)
	queue := []string{headCommit.MGitHash}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		if visited[current] {
			continue
		}
		
		commit, err := storage.GetCommit(current)
		if err != nil {
			fmt.Printf("Error getting commit %s: %s\n", current, err)
			continue
		}
		
		commits[current] = commit
		visited[current] = true
		
		for _, parent := range commit.ParentHashes {
			if !visited[parent] {
				queue = append(queue, parent)
			}
		}
	}
	
	// Verify each commit's hash
	valid := true
	fmt.Printf("Verifying %d MGit commits...\n", len(commits))
	
	for hash, commit := range commits {
		// Get the Git commit
		gitHash := commit.GitHash
		repo := getRepo()
		gitCommit, err := repo.CommitObject(plumbing.NewHash(gitHash))
		if err != nil {
			fmt.Printf("Error: Cannot find Git commit %s: %s\n", gitHash, err)
			valid = false
			continue
		}
		
		// Compute the expected MGit hash
		expectedHash := computeMGitHash(gitCommit, commit.ParentHashes, commit.Author.Pubkey)
		
		if expectedHash.String() != hash {
			fmt.Printf("Hash verification failed for commit %s:\n", hash)
			fmt.Printf("  Expected: %s\n", expectedHash.String())
			fmt.Printf("  Actual:   %s\n", hash)
			valid = false
		}
	}
	
	if valid {
		fmt.Println("MGit commit chain verification successful!")
	} else {
		fmt.Println("MGit commit chain verification failed!")
		os.Exit(1)
	}
}