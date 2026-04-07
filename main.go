package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	version = "1.2.0"
	repo    = "jimmyalcala/clean-sql"
)

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func selfUpdate() {
	fmt.Println("Checking for updates...")

	resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing release info: %v\n", err)
		os.Exit(1)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == version {
		fmt.Printf("Already up to date (v%s)\n", version)
		return
	}

	fmt.Printf("Updating v%s -> v%s\n", version, latest)

	// Find the right binary for this OS/arch
	assetName := fmt.Sprintf("clean-sql-%s-%s", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		fmt.Fprintf(os.Stderr, "No binary found for %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
	}

	// Download to temp file
	fmt.Printf("Downloading %s...\n", assetName)
	dlResp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading: %v\n", err)
		os.Exit(1)
	}
	defer dlResp.Body.Close()

	tmpFile, err := os.CreateTemp("", "clean-sql-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, dlResp.Body); err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting permissions: %v\n", err)
		os.Exit(1)
	}

	// Find where the current binary lives
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable path: %v\n", err)
		os.Exit(1)
	}

	// Try direct replace first, fall back to sudo
	if err := copyFile(tmpFile.Name(), execPath); err != nil {
		fmt.Println("Need sudo to replace binary...")
		cmd := exec.Command("sudo", "cp", tmpFile.Name(), execPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error replacing binary: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Updated to v%s\n", latest)
}

// MySQL/MariaDB reserved words that commonly appear as column names.
// Source: https://dev.mysql.com/doc/refman/8.0/en/keywords.html
var reservedWords = map[string]bool{
	"accessible": true, "add": true, "all": true, "alter": true, "analyze": true,
	"and": true, "as": true, "asc": true, "asensitive": true, "before": true,
	"between": true, "bigint": true, "binary": true, "blob": true, "both": true,
	"by": true, "call": true, "cascade": true, "case": true, "change": true,
	"char": true, "character": true, "check": true, "collate": true, "column": true,
	"condition": true, "constraint": true, "continue": true, "convert": true,
	"create": true, "cross": true, "current_date": true, "current_time": true,
	"current_timestamp": true, "current_user": true, "cursor": true, "database": true,
	"databases": true, "day_hour": true, "day_microsecond": true, "day_minute": true,
	"day_second": true, "dec": true, "decimal": true, "declare": true, "default": true,
	"delayed": true, "delete": true, "desc": true, "describe": true, "deterministic": true,
	"distinct": true, "distinctrow": true, "div": true, "double": true, "drop": true,
	"dual": true, "each": true, "else": true, "elseif": true, "enclosed": true,
	"escaped": true, "exists": true, "exit": true, "explain": true, "false": true,
	"fetch": true, "float": true, "float4": true, "float8": true, "for": true,
	"force": true, "foreign": true, "from": true, "fulltext": true, "generated": true,
	"get": true, "grant": true, "group": true, "having": true, "high_priority": true,
	"hour_microsecond": true, "hour_minute": true, "hour_second": true, "if": true,
	"ignore": true, "in": true, "index": true, "infile": true, "inner": true,
	"inout": true, "insensitive": true, "insert": true, "int": true, "int1": true,
	"int2": true, "int3": true, "int4": true, "int8": true, "integer": true,
	"interval": true, "into": true, "io_after_gtids": true, "io_before_gtids": true,
	"is": true, "iterate": true, "join": true, "key": true, "keys": true,
	"kill": true, "leading": true, "leave": true, "left": true, "like": true,
	"limit": true, "linear": true, "lines": true, "load": true, "localtime": true,
	"localtimestamp": true, "lock": true, "long": true, "longblob": true,
	"longtext": true, "loop": true, "low_priority": true, "master_bind": true,
	"master_ssl_verify_server_cert": true, "match": true, "maxvalue": true,
	"mediumblob": true, "mediumint": true, "mediumtext": true, "middleint": true,
	"minute_microsecond": true, "minute_second": true, "mod": true, "modifies": true,
	"natural": true, "not": true, "no_write_to_binlog": true, "null": true,
	"numeric": true, "on": true, "optimize": true, "option": true, "optionally": true,
	"or": true, "order": true, "out": true, "outer": true, "outfile": true,
	"partition": true, "precision": true, "primary": true, "procedure": true,
	"purge": true, "range": true, "read": true, "reads": true, "read_write": true,
	"real": true, "references": true, "regexp": true, "release": true, "rename": true,
	"repeat": true, "replace": true, "require": true, "resignal": true, "restrict": true,
	"return": true, "revoke": true, "right": true, "rlike": true, "schema": true,
	"schemas": true, "second_microsecond": true, "select": true, "sensitive": true,
	"separator": true, "set": true, "show": true, "signal": true, "smallint": true,
	"spatial": true, "specific": true, "sql": true, "sqlexception": true,
	"sqlstate": true, "sqlwarning": true, "sql_big_result": true,
	"sql_calc_found_rows": true, "sql_small_result": true, "ssl": true,
	"starting": true, "stored": true, "straight_join": true, "table": true,
	"terminated": true, "then": true, "tinyblob": true, "tinyint": true,
	"tinytext": true, "to": true, "trailing": true, "trigger": true, "true": true,
	"undo": true, "union": true, "unique": true, "unlock": true, "unsigned": true,
	"update": true, "usage": true, "use": true, "using": true, "utc_date": true,
	"utc_time": true, "utc_timestamp": true, "values": true, "varbinary": true,
	"varchar": true, "varcharacter": true, "varying": true, "virtual": true,
	"when": true, "where": true, "while": true, "with": true, "write": true,
	"xor": true, "year_month": true, "zerofill": true,
	// MariaDB additional reserved words
	"action": true, "bit": true, "date": true, "enum": true, "no": true,
	"text": true, "time": true, "timestamp": true, "year": true,
	// MySQL 8.0+ reserved words that are common column names
	"rank": true, "row": true, "rows": true, "groups": true, "system": true,
	"function": true, "window": true, "over": true, "recursive": true,
}

func isReserved(word string) bool {
	return reservedWords[strings.ToLower(word)]
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_'
}

// processSQL reads the entire SQL stream character-by-character using a state
// machine. It tracks whether we are inside a single-quoted string literal
// (across line boundaries) and fixes reserved-word column names in INSERT...SET
// assignment lists.
//
// State machine:
//   - NORMAL: outside any string/statement context we care about
//   - IN_STRING: inside a single-quoted SQL string literal
//   - EXPECT_COL: after a comma outside a string in a SET clause, expecting a column name
//
// The key insight: in INSERT...SET syntax, after the SET keyword and after each
// value's closing quote/NULL/number followed by a comma, the next identifier
// before '=' is always a column name. We detect " SET " outside strings to enter
// column-tracking mode, and track commas outside strings to find column names.
// looksLikeEndOfValue peeks ahead in the buffered reader to determine if we
// just hit the end of a string value. After seeing \' we need to know if '
// ended the string or was an escaped quote. We look for patterns that only
// appear outside strings: ',column_name=' (SET clause) or ';' (end of statement).
// All peeked bytes are unread so the reader position is unchanged.
func looksLikeEndOfValue(br *bufio.Reader) bool {
	peeked, _ := br.Peek(128)
	if len(peeked) == 0 {
		return true // EOF = end of value
	}
	i := 0
	// Skip past any closing parens — these can appear inside string values
	// (e.g. filter expressions like \'%hold\'em%\'))')
	for i < len(peeked) && peeked[i] == ')' {
		i++
	}
	if i >= len(peeked) {
		return false
	}
	// We ONLY trust the ',column_name=' pattern as proof we're outside a string.
	// Semicolons are NOT reliable — CSS inside HTML email templates has plenty
	// of them (e.g. url(\'image.png\');\n). Even ';\n' appears in CSS.
	if peeked[i] == ',' {
		j := i + 1
		// Skip spaces
		for j < len(peeked) && peeked[j] == ' ' {
			j++
		}
		// Read identifier (column name): letters, digits, underscores
		identStart := j
		hasUnderscore := false
		for j < len(peeked) && isIdentChar(peeked[j]) {
			if peeked[j] == '_' {
				hasUnderscore = true
			}
			j++
		}
		// Require underscore in column name — SQL columns use snake_case
		// (page_content, filter_id) while CSS properties use camelCase
		// (GradientType, startColorstr). This avoids CSS false positives.
		if j > identStart && hasUnderscore && j < len(peeked) && peeked[j] == '=' {
			// Verify what comes after '=' looks like a SQL value start:
			// ' (string), N (NULL), or digit. NOT \ (which is CSS like
			// endColorstr=\'#fff\' inside gradient() functions).
			if j+1 < len(peeked) {
				after := peeked[j+1]
				if after == '\'' || after == 'N' || (after >= '0' && after <= '9') || after == '-' {
					return true
				}
			}
		}
	}
	return false
}

func processSQL(reader io.Reader, writer io.Writer, disableFK bool, totalSize int64) (int, error) {
	br := bufio.NewReaderSize(reader, 256*1024)
	bw := bufio.NewWriterSize(writer, 256*1024)
	defer bw.Flush()

	if disableFK {
		bw.WriteString("SET FOREIGN_KEY_CHECKS=0;\n")
	}

	fixCount := 0
	bytesRead := int64(0)
	lastProgress := time.Time{}

	// State
	inString := false      // inside a single-quoted string
	inInsert := false      // seen INSERT keyword, waiting for SET
	inSetClause := false   // inside a SET column=value list
	expectCol := false     // next identifier should be treated as a column name
	afterComma := false    // just saw a comma outside a string in SET context

	// Buffer for accumulating potential identifiers
	var identBuf []byte

	flushIdent := func(nextByte byte) {
		if len(identBuf) == 0 {
			return
		}
		word := string(identBuf)
		identBuf = identBuf[:0]

		// Check if this identifier is followed by '=' — meaning it's a column name
		if nextByte == '=' && inSetClause && expectCol {
			if isReserved(word) {
				bw.WriteByte('`')
				bw.WriteString(word)
				bw.WriteByte('`')
				fixCount++
				return
			}
		}

		if !inString {
			upperWord := strings.ToUpper(word)
			if upperWord == "INSERT" {
				inInsert = true
			} else if upperWord == "SET" && inInsert {
				inSetClause = true
				expectCol = true
			}
		}

		bw.WriteString(word)
	}

	// Ringbuffer to detect INSERT keyword for context
	// We'll use a simpler approach: track SET keyword outside strings

	for {
		b, err := br.ReadByte()
		if err != nil {
			flushIdent(0)
			if totalSize > 0 {
				fmt.Fprintf(os.Stderr, "\rProcessing: 100%% | %d fixes\n", fixCount)
			}
			if err == io.EOF {
				if disableFK {
					bw.WriteString("\nSET FOREIGN_KEY_CHECKS=1;\n")
				}
				return fixCount, nil
			}
			return fixCount, err
		}

		bytesRead++
		if totalSize > 0 && time.Since(lastProgress) > 150*time.Millisecond {
			pct := float64(bytesRead) / float64(totalSize) * 100
			bar := int(pct / 2)
			fmt.Fprintf(os.Stderr, "\rProcessing: %3.0f%% [%-50s] %s/%s | %d fixes",
				pct, progressBar(bar), humanSize(bytesRead), humanSize(totalSize), fixCount)
			lastProgress = time.Now()
		}

		if inString {
			// Inside a single-quoted string literal
			if len(identBuf) > 0 {
				// Shouldn't happen, but flush just in case
				bw.Write(identBuf)
				identBuf = identBuf[:0]
			}

			if b == 0 {
				// Null byte inside string — strip it
				fixCount++
				continue
			} else if b == '\\' {
				// Escaped character — write backslash and next char
				next, err := br.ReadByte()
				if err != nil {
					bw.WriteByte(b)
					if err == io.EOF {
						return fixCount, nil
					}
					return fixCount, err
				}
				if next == '\\' {
					// Double backslash — peek to see if followed by quote
					peek, peekErr := br.ReadByte()
					if peekErr == nil && peek == '\'' {
						// \\' — check if this is a double-escaped quote
						// by looking ahead for SET clause patterns
						if looksLikeEndOfValue(br) {
							// Legitimate escaped backslash followed by end of string
							bw.WriteByte('\\')
							bw.WriteByte('\\')
							bw.WriteByte('\'')
							inString = false
						} else {
							// Double-escaped quote — fix \\' to \'
							bw.WriteByte('\\')
							bw.WriteByte('\'')
							fixCount++
						}
					} else {
						// \\ not followed by quote — write both backslashes
						bw.WriteByte('\\')
						bw.WriteByte('\\')
						if peekErr == nil {
							br.UnreadByte()
						}
					}
				} else if next == '\'' {
					// \' — could be escaped quote OR trailing backslash + end of string
					// To decide, peek further: if ',column_name=' or ';' follows, it's
					// end of string. Otherwise it's a legitimate escaped quote.
					if looksLikeEndOfValue(br) {
						// Trailing backslash in value + end of string
						bw.WriteByte('\\')
						bw.WriteByte('\\')
						bw.WriteByte('\'')
						inString = false
						fixCount++
					} else {
						// Genuine escaped quote — stay in string
						bw.WriteByte(b)
						bw.WriteByte(next)
					}
				} else {
					bw.WriteByte(b)
					bw.WriteByte(next)
				}
			} else if b == '\'' {
				// Could be end of string or '' escape
				bw.WriteByte(b)
				next, err := br.ReadByte()
				if err != nil {
					inString = false
					if err == io.EOF {
						return fixCount, nil
					}
					return fixCount, err
				}
				if next == '\'' {
					// '' escape — still in string
					bw.WriteByte(next)
				} else {
					// End of string
					inString = false
					// Put back the byte we peeked
					br.UnreadByte()
				}
			} else if b == '\n' {
				// Literal newline inside string — escape it so mysql client
				// doesn't lose track of string boundaries across lines
				bw.WriteByte('\\')
				bw.WriteByte('n')
				fixCount++
			} else if b == '\r' {
				// Carriage return inside string — escape it
				// Check for \r\n sequence
				next, err := br.ReadByte()
				if err == nil && next == '\n' {
					bw.WriteByte('\\')
					bw.WriteByte('r')
					bw.WriteByte('\\')
					bw.WriteByte('n')
				} else {
					bw.WriteByte('\\')
					bw.WriteByte('r')
					if err == nil {
						br.UnreadByte()
					}
				}
				fixCount++
			} else if b == '\t' {
				// Tab inside string — escape it for safety
				bw.WriteByte('\\')
				bw.WriteByte('t')
			} else {
				bw.WriteByte(b)
			}
			continue
		}

		// Outside a string literal
		if b == 0 {
			// Null byte outside string — strip it
			fixCount++
			continue
		}
		if isIdentChar(b) {
			identBuf = append(identBuf, b)
			continue
		}

		// Non-identifier character — flush any accumulated identifier
		flushIdent(b)

		switch b {
		case '\'':
			inString = true
			expectCol = false
			afterComma = false
			bw.WriteByte(b)

		case '=':
			// After writing column name (handled in flushIdent)
			bw.WriteByte(b)
			expectCol = false

		case ',':
			bw.WriteByte(b)
			if inSetClause {
				afterComma = true
				expectCol = true
			}

		case ';':
			// End of statement — reset state
			bw.WriteByte(b)
			inInsert = false
			inSetClause = false
			expectCol = false
			afterComma = false

		case '`':
			// Backtick-quoted identifier — copy through unchanged
			bw.WriteByte(b)
			for {
				next, err := br.ReadByte()
				if err != nil {
					if err == io.EOF {
						return fixCount, nil
					}
					return fixCount, err
				}
				bw.WriteByte(next)
				if next == '`' {
					break
				}
			}

		default:
			_ = afterComma
			bw.WriteByte(b)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `clean-sql v%s — Fix MySQL/MariaDB SQL dump files

Backtick-quotes reserved words used as column names in INSERT...SET statements,
which commonly cause ERROR 1064 syntax errors during import.

Usage:
  clean-sql <input.sql>                  Fix and write to <input>_clean.sql
  clean-sql <input.sql> -o <output.sql>  Fix and write to specified output
  clean-sql --check <input.sql>          Check only, report issues (no changes)

The original file is NEVER modified. Output always goes to a new file.

Options:
  -o <file>      Output file path
  --disable-fk   Wrap output with SET FOREIGN_KEY_CHECKS=0/1
  --check        Dry run: report number of fixes needed without writing
  --update       Self-update to the latest release
  --version      Show version
  -h, --help     Show this help

Examples:
  clean-sql db-backup.sql
  clean-sql db-backup.sql --disable-fk
  clean-sql db-backup.sql -o fixed.sql
  cat dump.sql | clean-sql - > fixed.sql
`, version)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	var inputFile string
	var outputFile string
	var checkOnly bool
	var disableFK bool

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			usage()
			os.Exit(0)
		case "--version":
			fmt.Printf("clean-sql v%s\n", version)
			os.Exit(0)
		case "--update":
			selfUpdate()
			os.Exit(0)
		case "-o":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: -o requires an output file path")
				os.Exit(1)
			}
			i++
			outputFile = args[i]
		case "-i":
			fmt.Fprintln(os.Stderr, "Error: -i (in-place) is no longer supported to protect original files.")
			fmt.Fprintln(os.Stderr, "Use: clean-sql <input.sql> — output goes to <input>_clean.sql")
			os.Exit(1)
		case "--check":
			checkOnly = true
		case "--disable-fk":
			disableFK = true
		default:
			if inputFile == "" {
				inputFile = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument: %s\n", args[i])
				os.Exit(1)
			}
		}
		i++
	}

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: no input file specified")
		os.Exit(1)
	}

	// Handle stdin
	if inputFile == "-" {
		fixCount, err := processSQL(os.Stdin, os.Stdout, disableFK, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Fixed %d reserved word column names\n", fixCount)
		os.Exit(0)
	}

	// Determine output path
	if outputFile == "" && !checkOnly {
		ext := ""
		base := inputFile
		if idx := strings.LastIndex(inputFile, "."); idx >= 0 {
			ext = inputFile[idx:]
			base = inputFile[:idx]
		}
		outputFile = base + "_clean" + ext
	}

	// Open input
	inFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input: %v\n", err)
		os.Exit(1)
	}
	defer inFile.Close()

	if checkOnly {
		fixCount, err := processSQL(inFile, io.Discard, false, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if fixCount == 0 {
			fmt.Println("No reserved word issues found.")
		} else {
			fmt.Printf("Found %d reserved word column names that need quoting.\n", fixCount)
		}
		os.Exit(0)
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
		os.Exit(1)
	}

	// Get file size for progress bar
	fileInfo, _ := inFile.Stat()
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	fixCount, err := processSQL(inFile, outFile, disableFK, fileSize)
	outFile.Close()
	inFile.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Fixed %d issues\n", fixCount)
	fmt.Fprintf(os.Stderr, "Output written to: %s\n", outputFile)
	fmt.Fprintf(os.Stderr, "Original file unchanged: %s\n", inputFile)
}

func progressBar(filled int) string {
	if filled > 50 {
		filled = 50
	}
	return strings.Repeat("=", filled) + strings.Repeat(" ", 50-filled)
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
