# git-auto

Automated git add-commit-push workflow with LLM-generated commit messages.

## Features

- Auto-stage changed files (all or untracked only)
- Generate conventional commit messages via OpenAI-compatible LLM API
- Interactive mode for file selection and commit message confirmation
- Customizable LLM prompt templates
- Configurable diff size limits for LLM input
- Intelligent diff summarization for large changesets
- **Security Layer 1**: Blocklist protection - auto-unstage sensitive files (.ssh/, .aws/, .env, *.pem, etc.)
- **Security Layer 2**: Redaction - mask API keys, tokens, passwords, and secrets before LLM processing
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

# Use intelligent summarization for large diffs (>10000 chars)
./git-auto --diff-threshold 10000

# Disable security checks (not recommended)
./git-auto --no-security
```

## Flags

- `-a`, `--all` - Stage all changed files (default: untracked only)
- `-m` - Commit message (if not provided, generates via LLM)
- `--dry-run` - Show what would be done without executing
- `--force-push` - Force push to remote (use with caution)
- `--interactive` - Interactive mode: select files to stage and confirm/edit commit message
- `--prompt` - Custom prompt template for LLM (use `%s` as placeholder for diff)
- `--max-diff` - Maximum characters of diff to send to LLM (0 = unlimited, default: 0)
- `--diff-threshold` - Character threshold for intelligent diff summarization (0 = disabled)
- `--no-security` - Disable security checks (blocklist and redaction)
- `--tag` - Create and push a tag after successful push

## Security Features

git-auto includes two layers of security protection that are **enabled by default**:

### Layer 1: Blocklist (Sanitizer)

Automatically detects and unstages sensitive files before they can be committed:

- `.ssh/` - SSH keys and configuration
- `.aws/` - AWS credentials and configuration
- `.env` - Environment files
- `*.pem` - Certificate and key files
- `id_rsa`, `id_dsa`, `id_ecdsa`, `id_ed25519` - Private key files
- `secrets.yaml` - Kubernetes/secrets files

**Behavior**: When sensitive files are detected in the staging area, they are automatically unstaged with a warning. The commit proceeds with the remaining safe files.

### Layer 2: Redaction

Before sending the diff to the LLM, sensitive content is masked:

- OpenAI API keys (`sk-...`)
- AWS Access Key IDs (`AKIA...`)
- GitHub tokens (`ghp_...`, `gho_...`)
- Bearer tokens
- Generic API keys, tokens, and passwords (e.g., `API_KEY=...`, `password=...`)
- Private key headers

**Behavior**: Sensitive patterns in the diff are replaced with `[REDACTED]` before being sent to the LLM. Your actual files on disk are never modified.

### Disabling Security

If you need to commit files that match these patterns (not recommended):

```bash
./git-auto --no-security
```

### Recommended .gitignore

Add these patterns to your `.gitignore` to prevent accidental staging:

```
.ssh/
.aws/
.env
*.pem
id_rsa*
id_dsa*
id_ecdsa*
id_ed25519*
secrets.yaml
```

## Intelligent Diff Summarization

For large changesets, git-auto can intelligently summarize the diff instead of truncating it:

```bash
# Use a 10000 character threshold
./git-auto --diff-threshold 10000
```

When the diff exceeds the threshold:
- Uses `git diff --stat` to generate a summary
- Lists all changed files
- Sends this summary to the LLM instead of the raw diff
- The LLM generates a commit message based on the summary

This approach:
- Preserves context better than simple truncation
- Works well with models that have limited context windows
- Configurable per model capabilities

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
2. **Security Check**: Blocklist check auto-unstages sensitive files (Layer 1)
3. **Redaction**: Sensitive content is masked in the diff (Layer 2)
4. **Diff Summarization**: If diff exceeds threshold, generates summary
5. **Commit Message**: If `-m` not provided, sends processed diff to LLM
6. **Commit**: Creates commit with the message
7. **Push**: Pushes to remote
8. **Handle Rejection**: If push is rejected (non-fast-forward):
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
