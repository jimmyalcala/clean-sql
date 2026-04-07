package main

import (
	"bytes"
	"strings"
	"testing"
)

func runProcess(t *testing.T, input string) (string, int) {
	t.Helper()
	reader := strings.NewReader(input)
	var buf bytes.Buffer
	count, err := processSQL(reader, &buf, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	return buf.String(), count
}

func TestFixReservedWordFrom(t *testing.T) {
	input := `INSERT IGNORE INTO email_custom SET id='1',from=NULL,attachments=NULL;`
	result, count := runProcess(t, input)

	if !strings.Contains(result, ",`from`=NULL") {
		t.Errorf("Expected backtick-quoted from, got: %s", result)
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestFixMultipleReservedWords(t *testing.T) {
	input := `INSERT INTO tbl SET from='x',key='y',order=NULL,name='z';`
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`='x'") {
		t.Errorf("Expected `from`, got: %s", result)
	}
	if !strings.Contains(result, "`key`='y'") {
		t.Errorf("Expected `key`, got: %s", result)
	}
	if !strings.Contains(result, "`order`=NULL") {
		t.Errorf("Expected `order`, got: %s", result)
	}
	// name is NOT a reserved word
	if strings.Contains(result, "`name`") {
		t.Errorf("name should NOT be quoted, got: %s", result)
	}
	if count != 3 {
		t.Errorf("Expected 3 fixes, got %d", count)
	}
}

func TestNoReservedWords(t *testing.T) {
	input := `INSERT INTO tbl SET username='foo',email='bar@baz.com';`
	result, count := runProcess(t, input)

	if result != input {
		t.Errorf("Expected no changes, got: %s", result)
	}
	if count != 0 {
		t.Errorf("Expected 0 fixes, got %d", count)
	}
}

func TestStringWithFromInside(t *testing.T) {
	// "from" inside a string value should NOT be quoted
	input := `INSERT INTO tbl SET body='select from table where from=1',from=NULL;`
	result, count := runProcess(t, input)

	// Only the column-name from should be quoted, not the one inside the string
	if !strings.Contains(result, ",`from`=NULL") {
		t.Errorf("Expected column from quoted, got: %s", result)
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestEscapedQuotesInString(t *testing.T) {
	input := `INSERT INTO tbl SET body='it\'s a test',from=NULL;`
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`=NULL") {
		t.Errorf("Expected from quoted, got: %s", result)
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestHTMLTemplateBody(t *testing.T) {
	input := `INSERT IGNORE INTO email_custom SET id='373453',subject='Hello',body='<html><body><p>Test</p></body></html>',from=NULL,attachments=NULL;`
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`=NULL") {
		t.Errorf("Expected from quoted, got: %s", result)
	}
	if strings.Contains(result, ",from=NULL") {
		t.Errorf("Found unquoted from=NULL in: %s", result)
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestNonInsertLinesUnchanged(t *testing.T) {
	input := "SELECT * FROM users;\nCREATE TABLE test (id INT);\n"
	result, count := runProcess(t, input)

	if result != input {
		t.Errorf("Expected no changes, got: %s", result)
	}
	if count != 0 {
		t.Errorf("Expected 0 fixes, got %d", count)
	}
}

func TestAlreadyQuoted(t *testing.T) {
	input := "INSERT INTO tbl SET `from`=NULL,id='1';"
	result, count := runProcess(t, input)

	if result != input {
		t.Errorf("Expected no changes, got: %s", result)
	}
	if count != 0 {
		t.Errorf("Expected 0 fixes, got %d", count)
	}
}

func TestMultiLineInsert(t *testing.T) {
	// INSERT with HTML body spanning multiple lines — newlines inside strings get escaped
	input := "INSERT IGNORE INTO email_custom SET id='1',body='<html>\n<body>\n<p>Hello</p>\n</body>\n</html>',from=NULL,attachments=NULL;"
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`=NULL") {
		t.Errorf("Expected from quoted in multi-line insert, got: %s", result)
	}
	// 4 newlines escaped + 1 reserved word fix = 5
	if count != 5 {
		t.Errorf("Expected 5 fixes (4 newlines + 1 reserved word), got %d", count)
	}
	// Newlines inside strings should be escaped
	if strings.Contains(result, "body='<html>\n") {
		t.Error("Expected literal newlines inside strings to be escaped")
	}
	if !strings.Contains(result, `body='<html>\n<body>`) {
		t.Errorf("Expected escaped newlines, got: %s", result)
	}
}

func TestMultiLineNewlineEscape(t *testing.T) {
	// Verify newlines in string values are escaped, but newlines outside strings are preserved
	input := "INSERT INTO tbl SET body='line1\nline2\nline3',id='1';\nSELECT 1;\n"
	result, count := runProcess(t, input)

	// 2 newlines inside the string should be escaped
	if count != 2 {
		t.Errorf("Expected 2 fixes, got %d", count)
	}
	// The newline between statements should be preserved
	if !strings.Contains(result, ";\nSELECT") {
		t.Errorf("Newlines outside strings should be preserved, got: %s", result)
	}
}

func TestCRLFInString(t *testing.T) {
	input := "INSERT INTO tbl SET body='line1\r\nline2',id='1';"
	result, count := runProcess(t, input)

	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
	if !strings.Contains(result, `\r\n`) {
		t.Errorf("Expected \\r\\n escape, got: %s", result)
	}
}

func TestRealWorldLine(t *testing.T) {
	input := `INSERT IGNORE INTO email_custom SET id='373453',custom_title='Vendor',customer_username='test',body='<html><head></head><body><p>Hello</p></body></html>',unsub='<br>',paramValue='',embargotime='0',location_id='0',custtypes=NULL,sendrate=NULL,modifiedtime='0',from=NULL,attachments=NULL,dailystarttime=NULL,dailyendtime=NULL,attach_contract='0',attach_quote='0',attach_invoice='0';`
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`=NULL") {
		t.Error("Expected `from`=NULL in output")
	}
	if strings.Contains(result, ",from=NULL") {
		t.Error("Found unquoted from=NULL in output")
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestMultipleStatements(t *testing.T) {
	input := "INSERT INTO t1 SET from=NULL;\nINSERT INTO t2 SET from='a';\nSELECT 1;"
	result, count := runProcess(t, input)

	if count != 2 {
		t.Errorf("Expected 2 fixes across statements, got %d", count)
	}
	_ = result
}

func TestDoubleQuoteEscape(t *testing.T) {
	// '' is an alternative escape for single quote in SQL
	input := `INSERT INTO tbl SET body='it''s ok',from=NULL;`
	result, count := runProcess(t, input)

	if !strings.Contains(result, "`from`=NULL") {
		t.Errorf("Expected from quoted with '' escape, got: %s", result)
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}

func TestDisableFK(t *testing.T) {
	input := "INSERT INTO tbl SET id='1',from=NULL;"
	reader := strings.NewReader(input)
	var buf bytes.Buffer
	count, err := processSQL(reader, &buf, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := buf.String()

	if !strings.HasPrefix(result, "SET FOREIGN_KEY_CHECKS=0;\n") {
		t.Errorf("Expected FK disable at start, got: %s", result[:50])
	}
	if !strings.HasSuffix(result, "SET FOREIGN_KEY_CHECKS=1;\n") {
		t.Errorf("Expected FK enable at end, got: %s", result[len(result)-50:])
	}
	if count != 1 {
		t.Errorf("Expected 1 fix, got %d", count)
	}
}
