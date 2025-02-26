# MGit

MGit is a specialized Git implementation designed for secure, self-custodial medical data management using Nostr public keys for authentication. Built initially as a Go wrapper around git operations, it provides enhanced functionality for managing medical records in a distributed, self-sovereign manner.

## Overview

MGit extends Git with the following key features:
- Nostr public key integration for commit attribution
- Enhanced authentication for medical record repositories
- Custom hash generation that incorporates Nostr public keys
- Server-side components for secure repository hosting
- Client tools for repository management

## Core Functionality

MGit supports these operations:
- `mgit init` - Initialize a new repository
- `mgit clone <url> [path]` - Clone a repository with Nostr authentication
- `mgit add <files...>` - Add files to staging
- `mgit commit -m <message>` - Commit staged changes with Nostr public key attribution
- `mgit push` - Push commits to remote
- `mgit pull` - Pull changes from remote
- `mgit status` - Show repository status
- `mgit show [commit]` - Show commit details and changes
- `mgit config` - Get and set configuration values

## Authentication

MGit uses a challenge-response authentication system based on Nostr keys:

1. The client requests a challenge from the server
2. The challenge is signed using the user's Nostr private key
3. The server verifies the signature and issues a JWT token
4. The token is used for subsequent repository operations

## Development Roadmap

### Current Implementation
- Go-based implementation using go-git
- Server components using Node.js
- Nostr authentication integration
- Basic repository operations

### Future Development Paths

#### Web-Based Client
- In-browser implementation using isomorphic-git
- Browser storage for repository data
- React-based UI for medical record management

#### Mobile Integration Strategy
- Native module approach using libgit2
- Custom C library (libmgit2) implementing MGit functionality
- React Native integration for iOS and Android
- Full offline support for medical record access

## Usage

### Configuration
```
$ mgit config --global user.name "Your Name"
$ mgit config --global user.email "your.email@example.com"
$ mgit config --global user.pubkey "npub..."
```

### Server Authentication
```
# Authenticate with the MGit server
# (Currently implemented through the web interface)
# This generates a JWT token stored in ~/.mgitconfig/tokens.json
```

### Repository Operations
```
# Clone a repository
$ mgit clone http://mgit-server.com/repo-name

# Add and commit changes
$ mgit add medical-record.json
$ mgit commit -m "Update medical record with new lab results"

# View repository information
$ mgit show
```

## Self-Custody of Medical Data

The primary goal of MGit is to enable patients to maintain self-custody of their medical records. By using Git's robust version control features combined with Nostr's cryptographic identity system, MGit provides:

1. Verifiable authorship of medical record changes
2. Complete history of medical record updates
3. Secure, distributed storage
4. Patient-controlled access to medical information

## License

MIT