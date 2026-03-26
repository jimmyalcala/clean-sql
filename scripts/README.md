# restore

A wrapper script that combines `clean-sql`, MySQL import, and file cleanup into a single command.

## What It Does

1. Runs `clean-sql` on the SQL dump file to fix reserved word issues
2. Imports the cleaned file into MySQL/MariaDB
3. Moves both the original and cleaned files to a `restored_sql/` folder

## Install

Copy the script to a directory in your PATH:

```bash
cp restore ~/.local/bin/restore
chmod +x ~/.local/bin/restore
```

## Configuration

The script reads MySQL credentials from environment variables. Add these to your `~/.zshrc` or `~/.bashrc`:

```bash
export DB_HOST=localhost
export DB_USER=myuser
export DB_PASS=mypassword
export DB_NAME=mydatabase
```

Or pass them inline:

```bash
DB_USER=myuser DB_PASS=mypass DB_NAME=mydb restore dump.sql
```

## Usage

```bash
restore db-backup-mysite-1774532776.sql
```

Output:

```
==> Cleaning SQL file...
Fixed 40 reserved word column names
Output written to: db-backup-mysite-1774532776_clean.sql
==> Importing into MySQL...
==> Import complete.
==> Files moved to restored_sql/
Done.
```

After running, both files end up in `restored_sql/` inside the same directory:

```
restored_sql/
  db-backup-mysite-1774532776.sql
  db-backup-mysite-1774532776_clean.sql
```

## Requirements

- `clean-sql` installed and available in your PATH
- `mysql` client installed
