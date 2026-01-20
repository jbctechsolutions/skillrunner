# File Context Permissions

## Overview

Skillrunner automatically detects file references in user queries and prompts for permission before accessing them. This prevents unintended data exposure and gives users control over what files are shared with LLMs.

## How It Works

### 1. Automatic Detection

When you mention files in your query, they're automatically detected:

```bash
# Bare filenames
sr ask doc-gen "Explain phase_executor.go"

# Relative paths
sr ask doc-gen "Review ./internal/app/main.go"

# Multiple files
sr ask code-review "Compare file1.go and file2.go"
```

### 2. Permission Prompt

Before accessing files, you'll see a permission request:

```
📄 File Context Request
───────────────────────
The skill wants to access 2 file(s):

  1. internal/application/workflow/phase_executor.go (8.2 KB)
  2. internal/application/workflow/executor_test.go (15.3 KB)

Allow access to these files? [Y/n/individual/show]
```

### 3. Response Options

**Y (Yes)** - Approve all files
```bash
> y
✓ Approved access to 2 file(s)
```

**n (No)** - Deny all files
```bash
> n
✗ Access denied
```

**individual** - Approve each file separately
```bash
> individual

Approving files individually:

  [1/2] internal/application/workflow/phase_executor.go (8.2 KB)
  Allow? [Y/n/show] y
  ✓ Approved

  [2/2] internal/application/workflow/executor_test.go (15.3 KB)
  Allow? [Y/n/show] n
  ✗ Denied

✓ Approved 1 of 2 file(s)
```

**show** - Preview file contents before deciding
```bash
> show

File Previews:

─── [1] internal/application/workflow/phase_executor.go ───
    1 │ // Package workflow provides the workflow executor
    2 │ package workflow
    3 │
    4 │ import (
    5 │     "context"
   ...
  ... (150 more lines)

Allow access to all shown files? [Y/n/individual]
```

## Auto-Approve Mode

Skip permission prompts with the `--yes` flag:

```bash
# Auto-approve all file access
sr ask doc-gen "Review main.go" --yes

# Short form
sr ask doc-gen "Review main.go" -y
```

**Use case**: CI/CD pipelines, automated scripts, trusted environments.

**Warning**: Only use auto-approve in trusted environments where file exposure is acceptable.

## Security Features

### Sensitive File Detection

The system warns you about potentially sensitive files:

```
📄 File Context Request
───────────────────────
The skill wants to access 1 file(s):

  1. .env (245 B)

⚠️  Warning: Detected potentially sensitive files (.env, credentials, etc.)

Allow access to these files? [Y/n/individual/show]
```

Sensitive patterns detected:
- `.env`, `.env.*`
- `credentials`, `secret`, `password`, `token`
- `api_key`, `apikey`, `private_key`
- `.pem`, `.key`, `id_rsa`, SSH keys

### Size Limits

Files larger than 1MB are rejected:

```
─── large.bin ───
[File too large: 5242880 bytes, limit: 1048576 bytes]
```

### Binary Detection

Binary files are detected and skipped:

```
─── binary.exe ───
[Binary file, content not shown]
```

## File Detection Patterns

### Pattern 1: Explicit Paths
- `./file.go`
- `path/to/file.go`
- `../relative/path.go`

### Pattern 2: Bare Filenames
- `main.go`
- `config.yaml`
- `test.json`

Supported extensions: `.go`, `.yaml`, `.yml`, `.json`, `.js`, `.ts`, `.py`, `.rb`, `.java`, `.rs`, `.md`, `.txt`

### Search Behavior
1. **Current directory first**: Fast path for common case
2. **Recursive search**: Up to 5 levels deep
3. **Skip patterns**: `.git`, `node_modules`, `vendor`, `target`, `build`
4. **Deduplication**: Same file mentioned different ways only shown once

## Examples

### Basic Usage
```bash
# Single file with prompt
sr ask doc-gen "Explain main.go"
> y

# Multiple files
sr ask code-review "Compare old.go and new.go"
> y

# Auto-approve
sr ask doc-gen "Explain main.go" --yes
```

### Selective Approval
```bash
sr ask "Review database.go and .env.local"
> individual
  [1/2] database.go (3.5 KB)
  Allow? [Y/n/show] y
  ✓ Approved

  [2/2] .env.local (127 B)
  Allow? [Y/n/show] n
  ✗ Denied
```

### Preview Before Approval
```bash
sr ask doc-gen "Explain utils.go"
> show
# (shows first 10 lines)
Allow access to all shown files? [Y/n/individual] y
```

## Best Practices

### ✅ Do
- Review the file list before approving
- Use individual approval for mixed trusted/sensitive files
- Use `--yes` only in trusted, automated environments
- Pay attention to sensitive file warnings

### ❌ Don't
- Auto-approve when querying unfamiliar codebases
- Ignore sensitive file warnings without review
- Share sensitive files (.env, credentials) with LLMs
- Approve files you haven't reviewed

## Troubleshooting

### File Not Detected
```bash
# Try explicit path
sr ask doc-gen "Review ./path/to/file.go"

# Check file exists
ls path/to/file.go

# Check working directory
pwd
```

### Too Many Files
```bash
# Use individual mode to select specific files
sr ask "Review *.go"
> individual

# Or use explicit names
sr ask "Review main.go and config.go"
```

### Permission Denied
```bash
# File access was denied
✗ file access denied: user denied file access

# Solution: Approve files or use --yes flag
sr ask doc-gen "Review main.go" --yes
```

## Implementation Details

### Components

**File Detector** (`file_detector.go`)
- Regex-based pattern matching
- Recursive file search
- Binary/size validation
- Path deduplication

**Permission Prompt** (`permission.go`)
- Interactive CLI prompts
- Sensitive file detection
- File preview functionality
- Individual approval workflow

**Integration** (`ask.go`, `run.go`)
- Transparent injection
- Error handling
- User feedback

### Testing

```bash
# Run file detector tests
go test ./internal/infrastructure/context -v

# Test permission prompts manually
echo 'y' | sr ask doc-gen "Review main.go"
echo 'n' | sr ask doc-gen "Review main.go"
echo 'individual' | sr ask doc-gen "Review main.go"
```

## Future Enhancements

Planned features:
- **Permission persistence**: Remember approved files per session
- **Wildcard patterns**: `*.go` approval
- **Path allow/deny lists**: Config-based rules
- **Audit logging**: Track file access history
- **Smart defaults**: Auto-approve project files, deny sensitive files
