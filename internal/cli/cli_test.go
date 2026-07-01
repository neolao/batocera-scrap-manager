package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute_NoArgs_PrintsUsage(t *testing.T) {
	var out bytes.Buffer

	code := Execute(nil, &out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "Usage:")
	}
}

func TestExecute_VersionFlag_PrintsVersion(t *testing.T) {
	var out bytes.Buffer

	code := Execute([]string{"--version"}, &out)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), version) {
		t.Errorf("output = %q, want it to contain %q", out.String(), version)
	}
}

func TestExecute_UnknownCommand_ReturnsErrorCode(t *testing.T) {
	var out bytes.Buffer

	code := Execute([]string{"bogus"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "unknown command") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "unknown command")
	}
}
