# Pila™ - Stack all the things!

## Multi-Merge

Pila's multi-merge feature allows you to merge multiple branches into a single target branch in a controlled, repeatable way. This is useful for testing combinations of features together or creating integration branches.

### Basic Usage

Create a multi-merge by specifying branches to merge into a target branch:

```bash
pila multi-merge --branch feature-1 --branch feature-2 --branch feature-3 --target integration
```

Or use the shorthand:

```bash
pila mm -B feature-1 -B feature-2 -B feature-3 -T integration
```

This will:

1. Create or reset the target branch to point to the main branch
2. Merge each branch in the specified order
3. Save a manifest (`.pila_multi_merge.yaml`) to track the merge state

### Subcommands

#### `continue` - Resume after resolving conflicts

If a merge conflict occurs, resolve it manually, stage the changes, then continue:

```bash
# Resolve conflicts in your editor
git add <resolved-files>
pila multi-merge continue
```

#### `abort` - Cancel the multi-merge

Abort the current multi-merge operation and reset the target branch:

```bash
pila multi-merge abort
```

#### `show` - Display current status

Show which branches have been merged and which are pending:

```bash
pila multi-merge show
```

Output example:

```plain
feature-1 Merged
feature-2 Merged
feature-3 Not merged
```

#### `redo` - Reapply all merges from scratch

Reload the manifest and reapply all merges from the beginning. This resets the target branch to the main branch
and re-merges everything:

```bash
pila multi-merge redo
```

This is useful when:

- Branches have been updated and you want to recreate the integration branch
- You want to test the merge order again from a clean state

#### `append` - Add branches to the end

Add new branches to the end of an existing multi-merge:

```bash
pila multi-merge append --branch feature-4 --branch feature-5
```

This will:

1. Load the existing manifest
2. Append the new branches to the end
3. Re-create the target branch and merge all branches (existing + new)

#### `prepend` - Add branches to the start

Add new branches to the beginning of an existing multi-merge:

```bash
pila multi-merge prepend --branch feature-0
```

This will:

1. Load the existing manifest
2. Prepend the new branches to the start
3. Re-create the target branch and merge all branches (new + existing)

### Typical Workflow

1. **Create a multi-merge:**

   ```bash
   pila mm -B feature-auth -B feature-api -B feature-ui -T integration
   ```

2. **If conflicts occur:**

   ```bash
   # Fix conflicts
   git add .
   pila mm continue
   ```

3. **Check status:**

   ```bash
   pila mm show
   ```

4. **Add more branches:**

   ```bash
   pila mm append -B feature-logging
   ```

5. **Recreate after branch updates:**

   ```bash
   pila mm redo
   ```

6. **Clean up:**

   ```bash
   pila mm abort  # If you want to cancel
   ```

### The Manifest

Pila stores the multi-merge state in `.pila_multi_merge.yaml`:

```yaml
main_sha: abc123...
target: integration
type: branches
references:
  - name: feature-auth
    merged: true
  - name: feature-api
    merged: true
  - name: feature-ui
    merged: false
```

This manifest is committed to the target branch when all merges complete successfully, providing a record of what was merged.

### Notes

- Order matters! Branches are merged in the order specified.
- The target branch is reset to the main branch at the start of each multi-merge operation.
- If a branch doesn't exist (locally or remotely), it will be skipped with a warning.
- Remote branches (e.g., `origin/feature-name`) are preferred over local branches.

## Hooks

Pila supports custom hooks that run automatically in response to certain events. Hooks are shell scripts placed in the `.pila.hooks.d` directory at the root of your repository.

### Available Hooks

#### `multi-merge-completed.sh`

Runs automatically after a multi-merge completes successfully (all branches merged without errors).

**Arguments:**

- `$1` - The name of the target branch

**Example:**

```bash
#!/bin/bash
# .pila.hooks.d/multi-merge-completed.sh

TARGET_BRANCH="$1"

echo "Multi-merge completed for branch: $TARGET_BRANCH"

if [[ "$TARGET_BRANCH" != "main" ]]; then
  changie merge -u "## [Unreleased] - $(date +%Y-%m-%d)"
  git add CHANGELOG.md
  git commit -m 'chore: Update changelog with unreleased changes'
fi

# Push the branch to remote
git push -f origin "$TARGET_BRANCH"

# Notify team
curl -X POST https://hooks.slack.com/... -d "{\"text\": \"Integration branch $TARGET_BRANCH ready for testing\"}"
```

### Setting Up Hooks

1. Create the hooks directory:

   ```bash
   mkdir -p .pila.hooks.d
   ```

2. Create your hook script:

   ```bash
   touch .pila.hooks.d/multi-merge-completed.sh
   chmod +x .pila.hooks.d/multi-merge-completed.sh
   ```

3. Edit the script with your desired automation

4. The hook will run automatically when the event occurs

### Notes

- Hooks must be executable (`chmod +x`)
- Hooks should include a shebang line (e.g., `#!/bin/bash`)
- If a hook exits with a non-zero status, Pila will display an error
- Hook output is printed to the console
- You can add `.pila.hooks.d` to your `.gitignore` for local-only hooks, or commit them to share with your team
