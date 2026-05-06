# Release Checker

A command-line tool for monitoring the latest release versions of applications from git repositories. Designed for automated cron jobs, it fetches tags, compares against stored state, and prints a simple notification when a new version is available.

## Features

- Monitors multiple applications from a single configuration
- Supports any git repository (GitHub, GitLab, Codeberg, custom servers)
- Automatically determines the highest version tag using `git ls-remote --sort=v:refname`
- Updates state automatically with timestamps when new versions are detected
- Retries on network failures (3 attempts with exponential backoff)
- Silent operation when no updates are available
- Exit code > 0 if any errors occur
- Lightweight: single binary, no external dependencies except git

## Installation

### From Source

```bash
git clone <repository-url>
cd release-checker
make build
# Binary is at: build/release-checker
```

Or install to `/usr/local/bin`:

```bash
make install
```

### Dependencies

- Go 1.21+ (for building)
- `git` command-line tool (for fetching tags)

## Configuration

Create a YAML configuration file. The tool searches for config in this order:

1. Path specified with `--config` flag
2. `$RELEASE_CHECKER_CONFIG` environment variable
3. `~/.config/release-checker/config.yaml`
4. `./config.yaml` (current directory)

### Config Format

```yaml
# Optional: override state file location (defaults to same dir as binary)
state_file: "/path/to/state.json"

# List of applications to monitor
apps:
  - name: "kubectl"
    # GitHub shorthand - expands to https://github.com/kubernetes/kubernetes.git
    repo: "kubernetes/kubernetes"
    enabled: true

  - name: "terraform"
    # Full HTTPS URL
    repo: "https://codeberg.org/ziglang/zig.git"
    enabled: true

  - name: "old-app"
    # Disabled apps are skipped
    enabled: false
```

### Repository Formats

The `repo` field supports three formats:

1. **GitHub shorthand** (recommended): `owner/repo`
   - Expands to `https://github.com/owner/repo.git`

2. **Full URL**: `https://example.com/owner/repo.git`
   - Used as-is (can also use `http://`)

3. **Host/path**: `example.com/owner/repo` or `git.example.com/team/project`
   - Auto-prepends `https://` → `https://example.com/owner/repo.git`

The tool automatically normalizes all formats to a complete HTTPS URL before calling `git ls-remote`.

## State File

The state file (JSON) stores the last known version for each app. Default location: same directory as the binary (e.g., `/usr/local/bin/release-checker-state.json`). Override via config's `state_file` field or `--state` flag.

Example state file:

```json
{
  "version": "1.0",
  "last_check": "2025-05-06T19:00:00Z",
  "apps": {
    "kubectl": {
      "current_version": "v1.30.0",
      "last_updated": "2025-05-05T08:00:00Z",
      "update_count": 5
    }
  }
}
```

The tool creates the state file automatically on first run if missing.

## Usage

### Basic

```bash
release-checker
```

Uses default config search paths.

### Specify Config

```bash
release-checker --config /etc/release-checker/config.yaml
```

### Override State File

```bash
release-checker --state /var/lib/release-checker/state.json
```

### Dry Run (Preview Only)

```bash
release-checker --dry-run
```

Shows which apps would be updated without modifying the state file.

### Verbose Mode

```bash
release-checker --verbose
```

Shows all checks (not just updates). Useful for debugging.

### Show Version

```bash
release-checker --version
```

## Output Format

**When a new version is detected:**
```
APP_NAME: old_version -> new_version
```

**When version unchanged:**
- Silent by default
- With `--verbose`: `[OK] APP_NAME: current=old, latest=new`

**When an error occurs:**
```
[ERROR] APP_NAME: <error message>
```

## Exit Codes

- `0` - Success, no errors (even if updates were found)
- `1` - At least one app failed to check (network error, invalid repo, etc.)

## Cron Job Example

```cron
# Check daily at 8:00 AM
0 8 * * * /usr/local/bin/release-checker --config /etc/release-checker/config.yaml 2>&1 | mail -s "Release Updates" user@example.com
```

## How It Works

1. Reads configuration file
2. Loads state from JSON file
3. For each enabled app:
   - Runs `git ls-remote --tags --sort=v:refname <repo_url>` to list tags
   - Picks the last (highest) tag from sorted output
   - Strips `refs/tags/` prefix
   - Strips leading `v` for consistency (e.g., `v1.2.3` → `1.2.3`)
   - Compares with stored version
   - If different: prints notification and updates state
4. Saves state to disk (atomic write via temp file + rename)

## Error Handling & Retries

On network failures (timeouts, DNS errors, rate limiting), the tool:

1. Retries up to 3 times
2. Waits with exponential backoff (1s, 2s, 4s)
3. Retries happen silently (no output)
4. After final failure: prints error to stderr, continues with other apps, exits with code >0

## Troubleshooting

### "git command failed" errors

Ensure the `git` binary is installed and in PATH. The tool uses `git ls-remote`, which does not require a local clone.

### "No tags found" error

The repository might not have any tags. Add tags to the repository or disable the app in config.

### Private repository

Access to private repositories requires authentication. For HTTPS URLs, configure git credentials (e.g., via credential helper or SSH). SSH URLs (`git@...`) are not supported; use HTTPS with credentials instead.

### State file corruption

If the state file becomes corrupted, the tool automatically creates a backup (`.bak.YYYYMMDD-HHMMSS`) and starts fresh.

## License

MIT
