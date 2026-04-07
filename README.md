# clean-sql

Automatically fix errors when restoring MySQL/MariaDB SQL dump files. No more manually editing massive SQL files — just run `clean-sql` and import cleanly.

## Errors Resolved

| Error | Code | Cause | Fix Applied |
|-------|------|-------|-------------|
| `You have an error in your SQL syntax` | ERROR 1064 (42000) | Reserved words (`from`, `key`, `order`, etc.) used as unquoted column names in `INSERT...SET` | Backtick-quotes the reserved word column names |
| `Unknown command '\'` / `PAGER set to stdout` | mysql client error | Multi-line string values with literal newlines cause the mysql client to lose track of string boundaries | Escapes literal newlines inside strings as `\n` |
| `Cannot delete or update a parent row: a foreign key constraint fails` | ERROR 1451 (23000) | `DELETE`/`UPDATE` blocked by foreign key references in child tables | Wraps SQL with `SET FOREIGN_KEY_CHECKS=0/1` (`--disable-fk`) |
| `ASCII '\0' appeared in the statement` | mysql client error | Null bytes embedded in string values or between statements | Strips null bytes from the SQL stream |
| `You have an error in your SQL syntax` (double-escaped quotes) | ERROR 1064 (42000) | Backup tools double-escape single quotes (`hold\\'em` instead of `hold\'em`), causing premature string termination | Detects and fixes double-escaped quotes back to single-escaped |
| `Unknown command '\'` (trailing backslash) | mysql client error | Values ending with a literal backslash (`Thanksgiving \`) where the dump writes `\'` instead of `\\'`, causing the parser to misread the closing quote as an escaped quote | Detects trailing backslash + end-of-string and properly escapes |

Have an error not listed here? [Open an issue](https://github.com/jimmyalcala/clean-sql/issues) and we'll add support for it.

## The Problem

Backup systems often generate SQL dumps with syntax that breaks on import. The most common issues:

**Reserved words as column names** — not backtick-quoted:

```
ERROR 1064 (42000) at line 1519: You have an error in your SQL syntax;
check the manual that corresponds to your MariaDB server version for the
right syntax to use near 'from=NULL,attachments=NULL,...' at line 1
```

The offending SQL:

```sql
INSERT IGNORE INTO email_custom SET id='1',subject='Hello',body='<html>...</html>',from=NULL,attachments=NULL;
--                                                                                  ^^^^ reserved word!
```

`clean-sql` fixes it automatically:

```sql
INSERT IGNORE INTO email_custom SET id='1',subject='Hello',body='<html>...</html>',`from`=NULL,attachments=NULL;
```

**Double-escaped quotes** — backup tools escape quotes twice:

```sql
-- Broken: \\' is interpreted as literal backslash + end of string
INSERT INTO tbl SET col='hold\\'em';
-- Fixed: \' is a properly escaped quote
INSERT INTO tbl SET col='hold\'em';
```

**Trailing backslash in values** — the dump writes `\'` instead of `\\'`:

```sql
-- Broken: \' is read as escaped quote, string never ends
INSERT INTO tbl SET col='Thanksgiving \',next_col=NULL;
-- Fixed: \\' is escaped backslash + closing quote
INSERT INTO tbl SET col='Thanksgiving \\',next_col=NULL;
```

**Foreign key constraint errors** on `DELETE` or `UPDATE`:

```
ERROR 1451 (23000): Cannot delete or update a parent row: a foreign key
constraint fails
```

Doing this manually on a 2,000,000-line dump file with HTML email templates spanning multiple lines? No thanks. `clean-sql` handles it all.

## Why I Made This

There was no existing tool to post-process a MySQL/MariaDB dump file and fix these common import errors. The MySQL ecosystem assumes you either use `--quote-names` at dump time (which doesn't help when you already have a broken dump), or you fix it by hand.

`clean-sql` fills that gap. It uses a character-level state machine that properly tracks single-quoted string literals — even across multi-line HTML templates with escaped quotes and CSS content — so it only fixes actual column names and escape sequences, never corrupting content inside string values.

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
# Fix and write to db-backup_clean.sql (original is never modified)
clean-sql db-backup.sql

# Fix and write to a specific output file
clean-sql db-backup.sql -o fixed.sql

# Disable foreign key checks during import (fixes ERROR 1451)
clean-sql db-backup.sql --disable-fk

# Dry run — just report how many issues found
clean-sql --check db-backup.sql

# Read from stdin, write to stdout
cat dump.sql | clean-sql - > fixed.sql

# Self-update to the latest version
clean-sql --update
```

The original file is **never modified**. Output always goes to a new `_clean.sql` file (or the path you specify with `-o`).

A progress bar is displayed when processing files, showing percentage complete, bytes processed, and fix count:

```
Processing:  45% [======================                            ] 1.1GB/2.4GB | 3,421 fixes
```

### Options

| Flag | Description |
|------|-------------|
| `-o <file>` | Output file path |
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

# Fix everything + disable foreign key checks
clean-sql db-backup-mysite-1774532776.sql --disable-fk
# Processing: 100% [==================================================] 2.4GB/2.4GB | 7,357,888 fixes
# Fixed 7357888 issues
# Output written to: db-backup-mysite-1774532776_clean.sql

# Import the cleaned file
mysql -hlocalhost -uuser -ppassword mydb < db-backup-mysite-1774532776_clean.sql
# Success — 0 errors
```

### Updating

```bash
clean-sql --update
# Checking for updates...
# Updating v1.1.0 -> v1.2.0
# Downloading clean-sql-darwin-arm64...
# Updated to v1.2.0
```

The `--update` flag checks GitHub for the latest release, downloads the correct binary for your OS/architecture, and replaces the current binary automatically.

## How It Works

`clean-sql` reads the SQL file as a byte stream using a character-level state machine that:

1. Tracks whether the current position is inside a single-quoted string literal
2. Handles escape sequences (`\'`, `\\'`, and `''`) correctly across line boundaries
3. Escapes literal newlines (`\n`, `\r\n`) and strips null bytes inside string values
4. Detects and fixes double-escaped quotes (`\\'` -> `\'`) from backup tool bugs
5. Detects trailing backslash values where `\'` should be `\\'` at end of string
6. Detects `INSERT...SET` statements and backtick-quotes column names that are MySQL/MariaDB reserved words
7. Displays a real-time progress bar with percentage, size, and fix count

The escape-fix heuristic uses a smart look-ahead that peeks at bytes after an ambiguous `\'` or `\\'` sequence. It only treats these as end-of-string when followed by a `,column_name=` pattern where the column name contains an underscore (snake_case) and the value after `=` starts with `'`, `N` (NULL), or a digit. This avoids false positives from CSS content inside HTML email templates (e.g., `gradient(startColorstr=\'#fff\', endColorstr=\'#000\')`).

This means it correctly handles:
- Multi-line `INSERT` statements (HTML email templates, CSS stylesheets, BBCode content)
- Escaped quotes inside string values (including nested CSS/JS with `\'`)
- Double-escaped quotes from backup tool bugs (`hold\\'em` -> `hold\'em`)
- Values ending with literal backslashes (`Thanksgiving \`)
- `from=` appearing inside string content (not touched)
- `SET FOREIGN_KEY_CHECKS` and other non-INSERT `SET` statements (not touched)
- Already backtick-quoted identifiers (not double-quoted)
- Multiple statements in sequence
- Null bytes embedded in data

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
