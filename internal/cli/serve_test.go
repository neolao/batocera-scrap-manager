package cli

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer usable from two goroutines at once, needed
// because serve keeps writing to out while the test reads what it printed.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// setServeConfig points the CLI at a temporary config holding a registry
// folder with one game, and returns that registry folder.
func setServeConfig(t *testing.T) string {
	t.Helper()
	withTempConfig(t)
	registryFolder := t.TempDir()

	var out bytes.Buffer
	Execute([]string{"config", "set-registry", registryFolder}, &out)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	return registryFolder
}

func TestExecute_ServeHelp_PrintsUsageWithoutRequiringAConfiguredRegistry(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		withTempConfig(t)
		var out bytes.Buffer

		code := Execute([]string{"serve", flag}, &out)

		if code != 0 {
			t.Errorf("serve %s exit code = %d, want 0 (output: %s)", flag, code, out.String())
		}
		if !strings.Contains(out.String(), "batocera-scrap-manager serve") {
			t.Errorf("serve %s output = %q, want the command's usage", flag, out.String())
		}
		if strings.Contains(out.String(), "error") {
			t.Errorf("serve %s output = %q, want no error", flag, out.String())
		}
	}
}

func TestExecute_Serve_RegistryNotConfigured_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"serve"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "registry not configured") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestExecute_Serve_InvalidArguments_ReportTheProblemAndPrintUsage(t *testing.T) {
	cases := map[string]struct {
		args       []string
		wantInText string
	}{
		"unknown flag":           {[]string{"serve", "--bogus"}, "bogus"},
		"missing addr value":     {[]string{"serve", "--addr"}, "addr"},
		"addr without a port":    {[]string{"serve", "--addr", "8080"}, ":8080"},
		"unexpected positional":  {[]string{"serve", "extra"}, "extra"},
		"unexpected positional2": {[]string{"serve", "--addr", "127.0.0.1:0", "extra"}, "extra"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			setServeConfig(t)
			var out bytes.Buffer

			code := Execute(tc.args, &out)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1 (output: %s)", code, out.String())
			}
			if !strings.Contains(out.String(), "error") {
				t.Errorf("output = %q, want an error line", out.String())
			}
			if !strings.Contains(out.String(), tc.wantInText) {
				t.Errorf("output = %q, want it to mention %q", out.String(), tc.wantInText)
			}
			if !strings.Contains(out.String(), "batocera-scrap-manager serve") {
				t.Errorf("output = %q, want the command's usage after the error", out.String())
			}
		})
	}
}

var listeningPattern = regexp.MustCompile(`listening on 127\.0\.0\.1:([1-9][0-9]*)`)

func TestExecute_Serve_ServesTheRegistryThenStopsCleanlyOnInterrupt(t *testing.T) {
	setServeConfig(t)
	out := &syncBuffer{}

	done := make(chan int, 1)
	go func() { done <- Execute([]string{"serve", "--addr", "127.0.0.1:0"}, out) }()

	address := waitForListeningAddress(t, out)
	if strings.Contains(out.String(), "http://localhost") {
		t.Errorf("output = %q, want no localhost hint for an explicit host", out.String())
	}

	resp, err := http.Get("http://" + address + "/")
	if err != nil {
		t.Fatalf("GET / on the running server: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Sonic") {
		t.Errorf("served page does not list the registry's game, got: %s", body)
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("failed to interrupt the server: %v", err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0 after an interrupt", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop within 5s of an interrupt")
	}
}

// waitForListeningAddress waits for serve to print the address it listens
// on, and returns it.
func waitForListeningAddress(t *testing.T, out *syncBuffer) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if match := listeningPattern.FindString(out.String()); match != "" {
			return strings.TrimPrefix(match, "listening on ")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("serve never printed the address it listens on, output: %s", out.String())
	return ""
}

func TestExecute_Serve_WildcardAddress_HintsAReachableBrowserURL(t *testing.T) {
	lines := listeningLines("0.0.0.0:8080")

	if !strings.Contains(lines, "listening on 0.0.0.0:8080") {
		t.Errorf("lines = %q, want the resolved listening address", lines)
	}
	if !strings.Contains(lines, "http://localhost:8080") {
		t.Errorf("lines = %q, want a browser URL that actually resolves", lines)
	}
	if strings.Contains(lines, "http://0.0.0.0:8080") {
		t.Errorf("lines = %q, want no unusable wildcard URL", lines)
	}
}

func TestExecute_Serve_ExplicitHost_PrintsThatHostAsTheBrowserURL(t *testing.T) {
	lines := listeningLines("192.168.1.10:9000")

	if !strings.Contains(lines, "listening on 192.168.1.10:9000") {
		t.Errorf("lines = %q, want the resolved listening address", lines)
	}
	if !strings.Contains(lines, "http://192.168.1.10:9000") {
		t.Errorf("lines = %q, want the browser URL of the requested host", lines)
	}
	if strings.Contains(lines, "localhost") {
		t.Errorf("lines = %q, want no localhost hint for an explicit host", lines)
	}
}

func TestExecute_Help_MentionsTheServeCommand(t *testing.T) {
	var out bytes.Buffer

	Execute([]string{"--help"}, &out)

	if !strings.Contains(out.String(), "serve") {
		t.Errorf("general help = %q, want it to list the serve command", out.String())
	}
}
