package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// HandleUploadPack handles the upload-pack command
// This is used by the server to serve Git repositories over HTTP
func HandleUploadPack(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit upload-pack [--stateless-rpc] <repository>")
		os.Exit(1)
	}

	// Check for --stateless-rpc flag
	statelessRPC := false
	repoPath := args[0]

	if args[0] == "--stateless-rpc" {
		statelessRPC = true
		if len(args) < 2 {
			fmt.Println("Usage: mgit upload-pack [--stateless-rpc] <repository>")
			os.Exit(1)
		}
		repoPath = args[1]
	}

	// Verify the repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Printf("Error: repository at %s does not exist\n", repoPath)
		os.Exit(1)
	}

	// Check if it has a .git directory (it's a proper Git repository)
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// If no .git directory, it might be a bare repository (the repo itself is the .git dir)
		if _, err := os.Stat(filepath.Join(repoPath, "objects")); os.IsNotExist(err) {
			fmt.Printf("Error: %s is not a valid Git repository\n", repoPath)
			os.Exit(1)
		}
		gitDir = repoPath
	}

	// Prepare Git upload-pack arguments
	gitArgs := []string{"upload-pack"}
	if statelessRPC {
		gitArgs = append(gitArgs, "--stateless-rpc")
	}
	gitArgs = append(gitArgs, repoPath)

	// Execute git-upload-pack
	cmd := exec.Command("git", gitArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing git upload-pack: %s\n", err)
		os.Exit(1)
	}

	// After the standard Git upload-pack, we could add custom MGit functionality
	// But for now, we're just forwarding to the standard Git command for compatibility
}