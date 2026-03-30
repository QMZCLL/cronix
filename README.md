# cronix

cronix is a Linux cron job manager that wraps the system crontab with a friendly CLI. It keeps your scheduled tasks isolated in `~/.config/cronix/tasks.json`, writes timestamped execution logs per task, and lets you manage everything without editing crontab files by hand.

> **Note**: A TUI interface (`cronix tui`) is planned but not yet implemented. All current functionality is available through the CLI subcommands documented below.

---

## Installation

### One-line installer (Linux amd64 / arm64)

```bash
curl -fsSL https://raw.githubusercontent.com/QMZCLL/cronix/main/install.sh | bash
```

The script detects your architecture, downloads the matching binary from GitHub Releases, installs it to `/usr/local/bin/cronix`, and verifies the installation.

### Build from source

Requirements: Go 1.21+

```bash
git clone https://github.com/QMZCLL/cronix.git
cd cronix
make build
# binary placed at dist/cronix
```

Cross-compile for Linux targets:

```bash
make build-linux-amd64   # -> dist/cronix-linux-amd64
make build-linux-arm64   # -> dist/cronix-linux-arm64
```

---

## Quick Start

```bash
# Initialize cronix (creates ~/.config/cronix/ directory structure)
cronix init

# Add a task that runs every day at 02:00
cronix add --name nightly-backup --cron "0 2 * * *" --cmd "/usr/local/bin/backup.sh" \
  --desc "Nightly backup"

# List all tasks
cronix list

# View today's logs for a task
cronix logs nightly-backup

# Manually run a task right now
cronix run nightly-backup
```

---

## CLI Reference

### `cronix init`

Initializes the cronix configuration directory (`~/.config/cronix/`), creates `tasks.json` if it does not exist, and injects the managed crontab block. Safe to run multiple times.

```
cronix init
```

---

### `cronix add`

Adds a new scheduled task and registers it with the system crontab.

```
cronix add --name <name> --cron <expr> --cmd <command> [flags]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--name` | yes | Unique task name (used as identifier and log directory name) |
| `--cron` | yes | Standard 5-field cron expression (e.g. `"0 2 * * *"`) |
| `--cmd` | yes | Shell command to execute |
| `--desc` | no | Human-readable description |
| `--env KEY=VALUE` | no | Environment variable; repeatable |

**Example**

```bash
cronix add --name ping-check --cron "*/5 * * * *" --cmd "ping -c1 8.8.8.8" \
  --env HTTP_PROXY=http://proxy:3128
```

---

### `cronix list`

Prints all registered tasks in a table with columns: `NAME`, `CRON`, `STATUS`, `COMMAND`.

```
cronix list
```

---

### `cronix enable`

Enables a disabled task and re-registers it with the system crontab.

```
cronix enable <name>
```

---

### `cronix disable`

Disables a task and removes it from the system crontab. The task definition is kept in `tasks.json`.

```
cronix disable <name>
```

---

### `cronix remove`

Permanently removes a task and deletes it from the system crontab.

```
cronix remove <name>
```

---

### `cronix run`

Manually executes a task immediately (outside of the cron schedule). Output is written to the log file for today.

```
cronix run <name>
```

---

### `cronix logs`

Displays the execution log for a task.

```
cronix logs <name> [--date YYYY-MM-DD] [--tail N]
```

| Flag | Description |
|------|-------------|
| `--date` | Show logs for a specific date (default: today) |
| `--tail N` | Show only the last N lines |

**Examples**

```bash
# Today's log
cronix logs nightly-backup

# Log for a specific date
cronix logs nightly-backup --date 2025-01-10

# Last 20 lines
cronix logs nightly-backup --tail 20
```

---

## Log Files

Execution logs are stored at:

```
~/cronix-logs/<task-name>/YYYY-MM-DD.log
```

Each log entry is wrapped with run headers:

```
=== Run at 2025-01-15T02:00:01+08:00 ===
... command output ...
=== Exit: 0 | Duration: 1.234s ===
```

### Custom log directory

Override the default log location via environment variable or config:

```bash
# Environment variable (takes priority)
export CRONIX_LOG_DIR=/var/log/cronix

# Or set log_dir in ~/.config/cronix/tasks.json
```

---

## Configuration

cronix stores its state in `~/.config/cronix/tasks.json`. This file is managed automatically -- do not edit it by hand.

```json
{
  "tasks": [
    {
      "name": "nightly-backup",
      "command": "/usr/local/bin/backup.sh",
      "cron_expr": "0 2 * * *",
      "enabled": true,
      "envs": {},
      "created_at": "2025-01-15T10:00:00Z",
      "last_run_at": null,
      "description": "Nightly backup"
    }
  ],
  "log_dir": ""
}
```

Wrapper scripts for each task are stored in `~/.config/cronix/wrappers/`. These are shell scripts that set up environment variables and redirect output to the log file. They are regenerated automatically by cronix.

cronix manages a dedicated block in your user crontab, delimited by:

```
# cronix-managed-start
...
# cronix-managed-end
```

Entries outside this block are never modified.

---

## Project Layout

```
cronix/
  cmd/cronix/       # CLI entry point and subcommands
  internal/
    config/         # Config load/save
    cron/           # crontab read/write, wrapper script generation
    logger/         # Log file management
    task/           # Task CRUD
  Makefile
  install.sh
```

---

## License

MIT
