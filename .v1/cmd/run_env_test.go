package main

import (
	"testing"
)

func TestParseEnvContent_EmptyInput(t *testing.T) {
	result, err := parseEnvContent("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestParseEnvContent_SingleKeyValue(t *testing.T) {
	result, err := parseEnvContent("KEY=value")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "value" {
		t.Fatalf("expected KEY=value, got %v", result)
	}
}

func TestParseEnvContent_MultipleLines(t *testing.T) {
	content := "KEY1=value1\nKEY2=value2\nKEY3=value3"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result["KEY1"] != "value1" || result["KEY2"] != "value2" || result["KEY3"] != "value3" {
		t.Fatalf("unexpected values: %v", result)
	}
}

func TestParseEnvContent_CommentsIgnored(t *testing.T) {
	content := "# This is a comment\nKEY=value\n  # Another comment"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result["KEY"] != "value" {
		t.Fatalf("expected KEY=value, got %v", result)
	}
}

func TestParseEnvContent_BlankLinesIgnored(t *testing.T) {
	content := "KEY1=value1\n\nKEY2=value2\n\n"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
}

func TestParseEnvContent_QuotedValues(t *testing.T) {
	content := `KEY1="quoted value"
KEY2='single quoted'
KEY3=unquoted`
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY1"] != "quoted value" {
		t.Fatalf("expected 'quoted value', got %q", result["KEY1"])
	}
	if result["KEY2"] != "single quoted" {
		t.Fatalf("expected 'single quoted', got %q", result["KEY2"])
	}
	if result["KEY3"] != "unquoted" {
		t.Fatalf("expected 'unquoted', got %q", result["KEY3"])
	}
}

func TestParseEnvContent_MissingEquals(t *testing.T) {
	content := "MISSING_EQUALS"
	_, err := parseEnvContent(content)
	if err == nil {
		t.Fatal("expected error for missing '='")
	}
}

func TestParseEnvContent_ExportPrefix(t *testing.T) {
	content := "export KEY=value"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "value" {
		t.Fatalf("expected KEY=value, got %v", result)
	}
}

func TestParseEnvContent_EmptyValue(t *testing.T) {
	content := "EMPTY="
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["EMPTY"] != "" {
		t.Fatalf("expected empty string, got %q", result["EMPTY"])
	}
}

func TestParseEnvContent_WindowsLineEndings(t *testing.T) {
	content := "KEY=value\r\n"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "value" {
		t.Fatalf("expected 'value', got %q", result["KEY"])
	}
}

func TestParseEnvContent_ValueWithEquals(t *testing.T) {
	content := "KEY=foo=bar"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "foo=bar" {
		t.Fatalf("expected 'foo=bar', got %q", result["KEY"])
	}
}

func TestParseEnvContent_EmptyKey(t *testing.T) {
	content := "=value"
	_, err := parseEnvContent(content)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestParseEnvContent_SpacesAroundKey(t *testing.T) {
	content := "  KEY  =value"
	result, err := parseEnvContent(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "value" {
		t.Fatalf("expected KEY=value, got %v", result)
	}
}
