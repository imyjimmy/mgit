package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		initRepo(args)
	case "clone":
		cloneRepo(args)
	case "add":
		addFiles(args)
	case "commit":
		commitChanges(args)
	case "push":
		pushChanges(args)
	case "pull":
		pullChanges(args)
	case "status":
		showStatus(args)
	case "branch":
		handleBranch(args)
	case "checkout":
		checkoutBranch(args)
	case "log":
		showLog(args)
	case "show":
		HandleShow(args)
	case "config":
		HandleConfig(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("mgit - A go-git wrapper")
	fmt.Println("Usage: mgit <command> [args]")
	fmt.Println("Commands:")
	fmt.Println("  init            Initialize a new repository")
	fmt.Println("  clone <url>     Clone a repository")
	fmt.Println("  add <files...>  Add files to staging")
	fmt.Println("  commit -m <msg> Commit staged changes")
	fmt.Println("  push            Push commits to remote")
	fmt.Println("  pull            Pull changes from remote")
	fmt.Println("  status          Show repository status")
	fmt.Println("  branch          List branches")
	fmt.Println("  branch <name>   Create a new branch")
	fmt.Println("  checkout <ref>  Checkout a branch or commit")
	fmt.Println("  log             Show commit history")
	fmt.Println("  show [commit]    Show commit details and changes")
	fmt.Println("  config          Get and set configuration values")
}

func initRepo(args []string) {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	_, err := git.PlainInit(path, false)
	if err != nil {
		fmt.Printf("Error initializing repository: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Initialized empty Git repository in %s\n", path)
}

func cloneRepo(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit clone <url> [path]")
		os.Exit(1)
	}

	url := args[0]
	path := "."
	if len(args) > 1 {
		path = args[1]
	}

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Printf("Error cloning repository: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cloned repository %s to %s\n", url, path)
}

func getRepo() *git.Repository {
	repo, err := git.PlainOpen(".")
	if err != nil {
		fmt.Printf("Error opening repository: %s\n", err)
		os.Exit(1)
	}
	return repo
}

func addFiles(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit add <files...>")
		os.Exit(1)
	}

	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	for _, file := range args {
		_, err := w.Add(file)
		if err != nil {
			fmt.Printf("Error adding file %s: %s\n", file, err)
			os.Exit(1)
		}
	}
	fmt.Println("Changes staged for commit")
}

func commitChanges(args []string) {
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

	// Use the custom MGitCommit function with MCommitOptions
	commit, err := MGitCommit(message, &MCommitOptions{
		Author: &Signature{
			Name:   GetConfigValue("user.name", "mgit User"),
			Email:  GetConfigValue("user.email", "mgit@example.com"),
			Pubkey: GetConfigValue("user.pubkey", ""),
			When:   time.Now(),
		},
	})
	if err != nil {
		fmt.Printf("Error committing changes: %s\n", err)
		os.Exit(1)
	}

	// Since we're using a custom hash, we need to handle how to display it
	// Option 1: Try to get the commit object (may not work with custom hash)
	repo := getRepo()
	obj, err := repo.CommitObject(commit)
	if err != nil {
		// Option 2: Just display the hash if we can't get the object
		fmt.Printf("Committed changes [%s]: %s\n", commit.String()[:7], message)
	} else {
		fmt.Printf("Committed changes [%s]: %s\n", obj.Hash.String()[:7], message)
	}
}

func pushChanges(args []string) {
	repo := getRepo()
	
	// Get authentication if provided through environment variables
	auth := &http.BasicAuth{
		Username: os.Getenv("MGIT_USERNAME"),
		Password: os.Getenv("MGIT_PASSWORD"),
	}

	// Don't use auth if credentials aren't provided
	var authOption *http.BasicAuth
	if auth.Username != "" && auth.Password != "" {
		authOption = auth
	}

	err := repo.Push(&git.PushOptions{
		Auth:     authOption,
		Progress: os.Stdout,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			fmt.Println("Everything up-to-date")
			return
		}
		fmt.Printf("Error pushing changes: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Changes pushed to remote")
}

func pullChanges(args []string) {
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	err = w.Pull(&git.PullOptions{
		Progress: os.Stdout,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			fmt.Println("Already up-to-date")
			return
		}
		fmt.Printf("Error pulling changes: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Changes pulled from remote")
}

func showStatus(args []string) {
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}

	status, err := w.Status()
	if err != nil {
		fmt.Printf("Error getting status: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Current branch:", getCurrentBranch(repo))
	fmt.Println()
	
	if status.IsClean() {
		fmt.Println("Nothing to commit, working tree clean")
		return
	}

	fmt.Println("Changes to be committed:")
	for file, fileStatus := range status {
		if fileStatus.Staging == git.Added {
			fmt.Printf("  new file:   %s\n", file)
		} else if fileStatus.Staging == git.Modified {
			fmt.Printf("  modified:   %s\n", file)
		} else if fileStatus.Staging == git.Deleted {
			fmt.Printf("  deleted:    %s\n", file)
		}
	}
	fmt.Println()

	fmt.Println("Changes not staged for commit:")
	for file, fileStatus := range status {
		if fileStatus.Worktree == git.Modified {
			fmt.Printf("  modified:   %s\n", file)
		} else if fileStatus.Worktree == git.Deleted {
			fmt.Printf("  deleted:    %s\n", file)
		}
	}
	fmt.Println()

	fmt.Println("Untracked files:")
	for file, fileStatus := range status {
		if fileStatus.Worktree == git.Untracked {
			fmt.Printf("  %s\n", file)
		}
	}
}

func getCurrentBranch(repo *git.Repository) string {
	head, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD: %s\n", err)
		return "unknown"
	}
	
	if head.Name().IsBranch() {
		return head.Name().Short()
	}
	
	return head.Hash().String()[:7]
}

func handleBranch(args []string) {
	repo := getRepo()
	
	if len(args) == 0 {
		// List branches
		branches, err := repo.Branches()
		if err != nil {
			fmt.Printf("Error listing branches: %s\n", err)
			os.Exit(1)
		}
		
		currentBranch := getCurrentBranch(repo)
		fmt.Println("Branches:")
		
		err = branches.ForEach(func(branch *plumbing.Reference) error {
			name := branch.Name().Short()
			if name == currentBranch {
				fmt.Printf("* %s\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error iterating branches: %s\n", err)
			os.Exit(1)
		}
	} else {
		// Create a new branch
		branchName := args[0]
		
		w, err := repo.Worktree()
		if err != nil {
			fmt.Printf("Error getting worktree: %s\n", err)
			os.Exit(1)
		}
		
		head, err := repo.Head()
		if err != nil {
			fmt.Printf("Error getting HEAD: %s\n", err)
			os.Exit(1)
		}
		
		err = w.Checkout(&git.CheckoutOptions{
			Hash:   head.Hash(),
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: true,
		})
		if err != nil {
			fmt.Printf("Error creating branch %s: %s\n", branchName, err)
			os.Exit(1)
		}
		
		fmt.Printf("Switched to a new branch '%s'\n", branchName)
	}
}

func checkoutBranch(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: mgit checkout <branch>")
		os.Exit(1)
	}
	
	repo := getRepo()
	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %s\n", err)
		os.Exit(1)
	}
	
	branchName := args[0]
	
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		// Maybe it's a commit hash?
		hash := plumbing.NewHash(branchName)
		err = w.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err != nil {
			fmt.Printf("Error checking out %s: %s\n", branchName, err)
			os.Exit(1)
		}
		fmt.Printf("Checked out commit %s\n", branchName)
	} else {
		fmt.Printf("Switched to branch '%s'\n", branchName)
	}
}

func showLog(args []string) {
	repo := getRepo()
	
	// Get the HEAD reference
	ref, err := repo.Head()
	if err != nil {
		fmt.Printf("Error getting HEAD: %s\n", err)
		os.Exit(1)
	}
	
	// Get commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		fmt.Printf("Error getting commit: %s\n", err)
		os.Exit(1)
	}
	
	// Get commit history
	commitIter, err := repo.Log(&git.LogOptions{From: commit.Hash})
	if err != nil {
		fmt.Printf("Error getting log: %s\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Commit History:")
	err = commitIter.ForEach(func(c *object.Commit) error {
		fmt.Printf("Commit: %s\n", c.Hash.String())
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		fmt.Printf("Date:   %s\n", c.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700"))
		fmt.Printf("\n    %s\n\n", c.Message)
		return nil
	})
	if err != nil {
		fmt.Printf("Error iterating commits: %s\n", err)
		os.Exit(1)
	}
}