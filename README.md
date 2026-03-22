# git-auto

Automated git add-commit-push workflow with LLM-generated commit messages.

## Features

- Auto-stage changed files (all or untracked only)
- Generate conventional commit messages via OpenAI-compatible LLM API
- Interactive mode for file selection and commit message confirmation
- Customizable LLM prompt templates
- Configurable diff size limits for LLM input
- Automatic push with merge strategy on rejection
- Conflict detection with user guidance

## Requirements

- Go 1.21+
- Git
- OpenAI-compatible API endpoint

## Installation

```bash
go build -o git-auto ./cmd/git-auto
```

Or install globally:

```bash
go install ./cmd/git-auto
```

## Configuration

Set environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | API key for LLM (required) | - |
| `OPENAI_BASE_URL` | API endpoint base URL | `https://api.openai.com/v1` |
| `OPENAI_MODEL` | Model to use | `gpt-3.5-turbo` |

## Usage

```bash
# Basic usage - stage untracked files, generate commit message, push
./git-auto

# Stage all changed files
./git-auto -a

# Interactive mode - select files and confirm commit message
./git-auto --interactive

# Provide custom commit message
./git-auto -m "feat: add new feature"

# Preview without executing
./git-auto --dry-run

# Force push (use with caution)
./git-auto --force-push

# Create and push a tag after successful push
./git-auto --tag v1.0.0

# Use custom LLM prompt
./git-auto --prompt "Generate a concise commit message: %s"

# Limit diff size sent to LLM (useful for large changes)
./git-auto --max-diff 10000
```

## Flags

- `-a`, `--all` - Stage all changed files (default: untracked only)
- `-m` - Commit message (if not provided, generates via LLM)
- `--dry-run` - Show what would be done without executing
- `--force-push` - Force push to remote (use with caution)
- `--interactive` - Interactive mode: select files to stage and confirm/edit commit message
- `--prompt` - Custom prompt template for LLM (use `%s` as placeholder for diff)
- `--max-diff` - Maximum characters of diff to send to LLM (0 = unlimited, default: 0)
- `--tag` - Create and push a tag after successful push

## Interactive Mode

When using `--interactive`, you can select files using:
- Single numbers: `1 3 5`
- Comma-separated: `1,3,5`
- Ranges: `1-4` (selects files 1 through 4)
- Combined: `1-4,6` (selects files 1-4 and 6)
- `all` - select all files
- `none` - cancel selection

## Workflow

1. **Stage**: Files are staged (all or untracked based on `-a` flag)
2. **Commit Message**: If `-m` not provided, sends staged diff to LLM to generate a conventional commit message
3. **Commit**: Creates commit with the message
4. **Push**: Pushes to remote
5. **Handle Rejection**: If push is rejected (non-fast-forward):
   - Automatically pulls with merge strategy
   - Retries push
   - If conflicts occur, reports them and exits for manual resolution

## Conflict Handling

If a merge conflict occurs:

1. The tool reports which files have conflicts
2. User must resolve conflicts manually: `git mergetool`
3. After resolution, run `git-auto` again

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Get detailed coverage by function
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## License

MIT
