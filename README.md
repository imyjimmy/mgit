# mgit

A Git wrapper built on top of go-git but allow nostr pubkeys and use it for medical data EMR transmission. This tool provides a command-line interface similar to Git but uses "mgit" as the command name instead of "git".

## Features

- Uses go-git for all Git operations (does not rely on system git)
- Provides a simple CLI for common Git operations
- Almost identical syntax to the standard git command

## Supported Commands

- `mgit init` - Initialize a new repository
- `mgit clone <url> [path]` - Clone a repository
- `mgit add <files...>` - Add files to staging
- `mgit commit -m <message>` - Commit staged changes
- `mgit push` - Push commits to remote
- `mgit pull` - Pull changes from remote
- `mgit status` - Show repository status
- `mgit branch` - List branches
- `mgit branch <name>` - Create a new branch
- `mgit checkout <branch/commit>` - Checkout a branch or commit
- `mgit log` - Show commit history

## Installation

```
go install github.com/yourusername/mgit@latest
```

Or build from source:

```
git clone https://github.com/yourusername/mgit.git
cd mgit
go build
```

## Configuration

Some configuration options can be set via environment variables:

- `MGIT_USER_NAME` - Username for commits (alternative to git config user.name)
- `MGIT_USER_EMAIL` - Email for commits (alternative to git config user.email)
- `MGIT_USERNAME` - Username for authentication with remote repositories
- `MGIT_PASSWORD` - Password/token for authentication with remote repositories

## Example Usage

```
# Initialize a new repository
mgit init

# Add files
mgit add main.go

# Commit changes
mgit commit -m "Initial commit"

# Create and switch to a new branch
mgit branch feature-branch
mgit checkout feature-branch

# View status
mgit status

# Push changes
mgit push
```

## Limitations

- Not all Git commands are implemented
- Some advanced features may be missing
- Authentication handling is simplified (uses environment variables instead of credential store)

## License

MIT