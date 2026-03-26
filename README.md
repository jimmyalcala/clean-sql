# clean-sql

Fix MySQL/MariaDB SQL dump files that fail to import with **ERROR 1064** (reserved words as column names) and **ERROR 1451** (foreign key constraints) during restore.

## The Problem

Many backup systems generate SQL dumps using `INSERT...SET` syntax where column names are not backtick-quoted. When a column name happens to be a MySQL/MariaDB reserved word (like `from`, `key`, `order`, `group`, etc.), the import fails:

```
ERROR 1064 (42000) at line 1519: You have an error in your SQL syntax;
check the manual that corresponds to your MariaDB server version for the
right syntax to use near 'from=NULL,attachments=NULL,...' at line 1
```

The offending SQL looks like this:

```sql
INSERT IGNORE INTO email_custom SET id='1',subject='Hello',body='<html>...</html>',from=NULL,attachments=NULL;
--                                                                                  ^^^^ reserved word!
```

The fix is simple — backtick-quote the column name:

```sql
INSERT IGNORE INTO email_custom SET id='1',subject='Hello',body='<html>...</html>',`from`=NULL,attachments=NULL;
```

But doing this manually on a 300,000-line dump file with HTML email templates spanning multiple lines? No thanks.

On top of that, imports often fail with **ERROR 1451** when `DELETE` or `UPDATE` statements hit foreign key constraints:

```
ERROR 1451 (23000): Cannot delete or update a parent row: a foreign key
constraint fails (`io`.`inoff_phonenumbers`, CONSTRAINT `phonenumber_greetingid`
FOREIGN KEY (`phonenumber_greetingid`) REFERENCES `inoff_phonerecordings`
(`phonerecording_id`) ON DELETE NO ACTION ON UPDATE NO ACTION)
```

`clean-sql` handles both problems.

## Why I Made This

There was no existing tool to post-process a MySQL/MariaDB dump file and fix reserved-word column names. The MySQL ecosystem assumes you either use `--quote-names` at dump time (which doesn't help when you already have a broken dump), or you fix it by hand.

`clean-sql` fills that gap. It uses a character-level state machine that properly tracks single-quoted string literals — even across multi-line HTML templates with escaped quotes — so it only fixes actual column names, never touching content inside string values.

## Install

### Quick Install (macOS & Linux)

```bash
curl -sSL https://raw.githubusercontent.com/jimmyalcala/clean-sql/main/install.sh | sh
```

Auto-detects your OS and architecture (macOS/Linux, arm64/amd64), downloads the right binary, and installs it to `/usr/local/bin`.

### From Source

Requires [Go](https://go.dev/dl/) 1.21+:

```bash
git clone https://github.com/jimmyalcala/clean-sql.git
cd clean-sql
make install
```

### Manual Download

Grab the binary for your platform from the [Releases](https://github.com/jimmyalcala/clean-sql/releases) page, make it executable, and move it to your PATH:

```bash
chmod +x clean-sql-darwin-arm64
sudo mv clean-sql-darwin-arm64 /usr/local/bin/clean-sql
```

## Usage

```bash
# Fix and write to db-backup_clean.sql
clean-sql db-backup.sql

# Fix and write to a specific output file
clean-sql db-backup.sql -o fixed.sql

# Fix the file in-place (overwrites original)
clean-sql db-backup.sql -i

# Disable foreign key checks during import (fixes ERROR 1451)
clean-sql db-backup.sql --disable-fk

# Dry run — just report how many issues found
clean-sql --check db-backup.sql

# Read from stdin, write to stdout
cat dump.sql | clean-sql - > fixed.sql

# Self-update to the latest version
clean-sql --update
```

### Options

| Flag | Description |
|------|-------------|
| `-o <file>` | Output file path |
| `-i` | Edit file in-place (overwrites original) |
| `--disable-fk` | Wrap output with `SET FOREIGN_KEY_CHECKS=0/1` to prevent foreign key errors |
| `--check` | Dry run: report number of fixes needed without writing |
| `--update` | Self-update to the latest release from GitHub |
| `--version` | Show version |
| `-h, --help` | Show help |

### Example Workflow

```bash
# Check how many issues exist
clean-sql --check db-backup-mysite-1774532776.sql
# Found 40 reserved word column names that need quoting.

# Fix reserved words + disable foreign key checks
clean-sql db-backup-mysite-1774532776.sql --disable-fk
# Fixed 40 reserved word column names
# Output written to: db-backup-mysite-1774532776_clean.sql

# Import the cleaned file
mysql -hlocalhost -uuser -ppassword mydb < db-backup-mysite-1774532776_clean.sql
# Success — 0 errors
```

### Updating

```bash
clean-sql --update
# Checking for updates...
# Updating v1.0.0 -> v1.1.0
# Downloading clean-sql-darwin-arm64...
# Updated to v1.1.0
```

The `--update` flag checks GitHub for the latest release, downloads the correct binary for your OS/architecture, and replaces the current binary automatically.

## How It Works

`clean-sql` reads the SQL file as a byte stream using a state machine that:

1. Tracks whether the current position is inside a single-quoted string literal
2. Handles escape sequences (`\'` and `''`) correctly across line boundaries
3. Detects `SET` keywords outside of strings to identify column assignment contexts
4. Backtick-quotes column names that are MySQL/MariaDB reserved words

This means it correctly handles:
- Multi-line `INSERT` statements (HTML email templates with newlines)
- Escaped quotes inside string values
- `from=` appearing inside string content (not touched)
- Already backtick-quoted identifiers (not double-quoted)
- Multiple statements in sequence

With `--disable-fk`, it also wraps the entire output with:

```sql
SET FOREIGN_KEY_CHECKS=0;
-- ... your SQL statements ...
SET FOREIGN_KEY_CHECKS=1;
```

This prevents foreign key constraint errors (ERROR 1451) when `DELETE` or `UPDATE` statements reference rows in child tables.

## Reporting Issues

Found a bug or have a suggestion? Please open an issue:

https://github.com/jimmyalcala/clean-sql/issues

When reporting a bug, include:
- The error message you see
- Your MariaDB/MySQL version
- A small sample of the SQL that causes the problem (redact sensitive data)

## Contributing

Contributions are welcome! Here's how:

1. Fork the repository
2. Create a feature branch: `git checkout -b my-feature`
3. Make your changes
4. Run the tests: `make test`
5. Commit and push: `git push origin my-feature`
6. Open a Pull Request

Please make sure all tests pass before submitting. If you're adding support for a new SQL pattern, include a test case.

### Development

```bash
# Run tests
make test

# Build locally
make build

# Cross-compile for all platforms
make release
```

## Thank You

If this tool saved you from manually editing a massive SQL dump file, consider giving it a star on GitHub. It helps others find it and keeps me motivated to maintain it.

## License

MIT
