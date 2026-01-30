# GoGit - Git Implementation from Scratch

A functional Git clone built from scratch in Go, demonstrating deep understanding of Git internals including content-addressable storage, cryptographic hashing, compression, and version control concepts.

## Features

### Implemented Commands

| Command | Description |
|---------|-------------|
| `gogit init` | Initialize a new repository |
| `gogit hash-object [-w] <file>` | Compute object hash, optionally write to database |
| `gogit cat-file [-p\|-t\|-s] <hash>` | Display object content, type, or size |
| `gogit add <files...>` | Stage files for commit |
| `gogit status` | Show working tree status |
| `gogit commit -m <message>` | Record changes to repository |
| `gogit log [--oneline]` | Show commit history |
| `gogit branch [name]` | List or create branches |
| `gogit checkout <ref>` | Switch branches or commits |
| `gogit diff` | Show changes between working tree and index |

### Git Internals Implemented

- **Object Model**: Blob, Tree, and Commit objects
- **Content-Addressable Storage**: SHA-1 hashing for object identification
- **Compression**: zlib compression for object storage
- **Index/Staging Area**: Binary index file format
- **References**: HEAD, branches, and symbolic refs
- **Diff Algorithm**: Line-based diff with unified format output

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/gogit.git
cd gogit

# Build
make build

# Or install to GOPATH/bin
make install
```

## Quick Start

```bash
# Initialize a new repository
gogit init

# Create and add a file
echo "Hello, GoGit!" > hello.txt
gogit add hello.txt

# Check status
gogit status

# Commit changes
gogit commit -m "Initial commit"

# View history
gogit log

# Create a branch
gogit branch feature

# Switch branches
gogit checkout feature

# Make changes and see diff
echo "More content" >> hello.txt
gogit diff
```

## Project Structure

```
gogit/
├── cmd/
│   └── gogit/
│       └── main.go              # Entry point
├── internal/
│   ├── commands/                # CLI commands (Cobra)
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── add.go
│   │   ├── commit.go
│   │   ├── log.go
│   │   ├── status.go
│   │   ├── branch.go
│   │   ├── checkout.go
│   │   ├── diff.go
│   │   ├── cat_file.go
│   │   └── hash_object.go
│   ├── object/                  # Git objects
│   │   ├── object.go
│   │   ├── blob.go
│   │   ├── tree.go
│   │   └── commit.go
│   ├── repository/              # Repository operations
│   │   ├── repository.go
│   │   └── refs.go
│   ├── index/                   # Staging area
│   │   └── index.go
│   ├── diff/                    # Diff algorithm
│   │   └── diff.go
│   └── utils/                   # Utilities
│       ├── hash.go
│       └── compress.go
├── go.mod
├── Makefile
└── README.md
```

## How It Works

### Object Storage

GoGit stores objects the same way Git does:

1. Create header: `<type> <size>\0`
2. Concatenate header + content
3. Compute SHA-1 hash
4. Compress with zlib
5. Store at `.gogit/objects/<first2>/<rest38>`

```go
// Example: Hashing a blob
header := fmt.Sprintf("blob %d\x00", len(content))
store := append([]byte(header), content...)
hash := sha1.Sum(store)
```

### Index Format

The staging area uses a binary format:

```
DIRC                    # Signature
<version>               # 4 bytes, big-endian
<entry count>           # 4 bytes, big-endian
<entries>               # Variable length
  - ctime, mtime        # Timestamps
  - dev, ino            # Device and inode
  - mode                # File mode
  - uid, gid            # Owner
  - size                # File size
  - sha1                # 20 bytes
  - flags               # 2 bytes
  - path                # Null-terminated
  - padding             # To 8-byte boundary
<checksum>              # SHA-1 of all above
```

### Commit Format

```
tree <tree-sha1>
parent <parent-sha1>
author <name> <email> <timestamp> <tz>
committer <name> <email> <timestamp> <tz>

<message>
```

## Verification

GoGit produces objects compatible with real Git:

```bash
# Create file with gogit
echo "test" > test.txt
gogit hash-object test.txt
# Output: 9daeafb9864cf43055ae93beb0afd6c7d144bfa4

# Verify with git
git hash-object test.txt
# Output: 9daeafb9864cf43055ae93beb0afd6c7d144bfa4
```

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run demo
make demo
```

## Limitations

This is an educational implementation. Notable limitations:

- No packfile support (loose objects only)
- No remote operations (push, pull, fetch, clone)
- No merge/rebase functionality
- Simplified tree handling (flat structure)
- No submodule support
- No hooks

## Learning Resources

- [Git Internals](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
- [Write yourself a Git](https://wyag.thb.lt/)
- [Git from the Bottom Up](https://jwiegley.github.io/git-from-the-bottom-up/)

## License

MIT License

## Author

Built as a learning project to understand Git internals.
