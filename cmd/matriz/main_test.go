package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "matriz") {
		t.Errorf("expected version output to contain 'matriz', got: %s", stdout.String())
	}
}

func TestCLI_VersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"version", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"version":`) {
		t.Errorf("expected JSON output with 'version', got: %s", stdout.String())
	}
}

func TestCLI_Doctor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "MATRIZ Doctor") {
		t.Errorf("expected doctor output to contain 'MATRIZ Doctor', got: %s", stdout.String())
	}
}

func TestCLI_DoctorJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doctor", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"checks":`) {
		t.Errorf("expected JSON output with 'checks', got: %s", stdout.String())
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"unknown-cmd"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for unknown command")
	}
	if !strings.Contains(stderr.String(), "Unknown command") {
		t.Errorf("expected stderr to report unknown command, got: %s", stderr.String())
	}
}
